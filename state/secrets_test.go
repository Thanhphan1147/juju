// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state_test

import (
	"fmt"
	"sort"
	"time"

	"github.com/juju/charm/v9"
	"github.com/juju/errors"
	"github.com/juju/names/v4"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/core/secrets"
	"github.com/juju/juju/core/watcher"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/testing"
	"github.com/juju/juju/testing/factory"
)

type SecretsSuite struct {
	testing.StateSuite
	store     state.SecretsStore
	owner     *state.Application
	ownerUnit *state.Unit
	relation  *state.Relation
}

var _ = gc.Suite(&SecretsSuite{})

func (s *SecretsSuite) SetUpTest(c *gc.C) {
	s.StateSuite.SetUpTest(c)
	s.store = state.NewSecrets(s.State)
	s.owner = s.Factory.MakeApplication(c, nil)
	s.ownerUnit = s.Factory.MakeUnit(c, &factory.UnitParams{Application: s.owner})
	app2 := s.Factory.MakeApplication(c, &factory.ApplicationParams{
		Name:  "wordpress",
		Charm: s.Factory.MakeCharm(c, &factory.CharmParams{Name: "wordpress"}),
	})
	ep1, err := s.owner.Endpoint("server")
	c.Assert(err, jc.ErrorIsNil)
	ep2, err := app2.Endpoint("db")
	c.Assert(err, jc.ErrorIsNil)
	s.relation = s.Factory.MakeRelation(c, &factory.RelationParams{
		Endpoints: []state.Endpoint{ep1, ep2},
	})
}

func ptr[T any](v T) *T {
	return &v
}

func (s *SecretsSuite) TestCreate(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	expire := now.Add(time.Hour).Round(time.Second).UTC()
	p := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Description:    ptr("my secret"),
			Label:          ptr("foobar"),
			ExpireTime:     ptr(expire),
			Params:         nil,
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, p)
	c.Assert(err, jc.ErrorIsNil)
	mc := jc.NewMultiChecker()
	mc.AddExpr(`_.CreateTime`, jc.Almost, jc.ExpectedValue)
	mc.AddExpr(`_.UpdateTime`, jc.Almost, jc.ExpectedValue)
	c.Assert(md, mc, &secrets.SecretMetadata{
		URI:              uri,
		Version:          1,
		Description:      "my secret",
		Label:            "foobar",
		RotatePolicy:     secrets.RotateDaily,
		NextRotateTime:   ptr(next),
		LatestRevision:   1,
		LatestExpireTime: ptr(expire),
		OwnerTag:         s.owner.Tag().String(),
		CreateTime:       now,
		UpdateTime:       now,
	})

	p.Label = nil
	_, err = s.store.CreateSecret(uri, p)
	c.Assert(err, jc.Satisfies, errors.IsAlreadyExists)
}

func (s *SecretsSuite) TestCreateProviderId(c *gc.C) {
	uri := secrets.NewURI()
	p := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			ProviderId:  ptr("provider-id"),
		},
	}
	_, err := s.store.CreateSecret(uri, p)
	c.Assert(err, jc.ErrorIsNil)
	v, providerId, err := s.store.GetSecretValue(uri, 1)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(v.EncodedValues(), gc.HasLen, 0)
	c.Assert(providerId, gc.NotNil)
	c.Assert(*providerId, gc.Equals, "provider-id")
}

func (s *SecretsSuite) TestCreateDuplicateLabel(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	expire := now.Add(time.Hour).Round(time.Second).UTC()
	p := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Description:    ptr("my secret"),
			Label:          ptr("foobar"),
			ExpireTime:     ptr(expire),
			Params:         nil,
			Data:           map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, p)
	c.Assert(err, jc.ErrorIsNil)
	uri2 := secrets.NewURI()
	_, err = s.store.CreateSecret(uri2, p)
	c.Assert(errors.Is(err, state.LabelExists), jc.IsTrue)
}

func (s *SecretsSuite) TestCreateDyingOwner(c *gc.C) {
	err := s.owner.Destroy()
	c.Assert(err, jc.ErrorIsNil)

	uri := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err = s.store.CreateSecret(uri, cp)
	c.Assert(err, gc.ErrorMatches, `cannot create secret for owner "application-mysql" which is not alive`)
}

func (s *SecretsSuite) TestGetValueNotFound(c *gc.C) {
	uri, _ := secrets.ParseURI("secret:9m4e2mr0ui3e8a215n4g")
	_, _, err := s.store.GetSecretValue(uri, 666)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
}

func (s *SecretsSuite) TestGetValue(c *gc.C) {
	uri := secrets.NewURI()
	p := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, p)
	c.Assert(err, jc.ErrorIsNil)

	val, providerId, err := s.store.GetSecretValue(md.URI, 1)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(providerId, gc.IsNil)
	c.Assert(val.EncodedValues(), jc.DeepEquals, map[string]string{
		"foo": "bar",
	})
}

func (s *SecretsSuite) TestListByOwner(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	expire := now.Add(time.Hour).Round(time.Second).UTC()
	p := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Description:    ptr("my secret"),
			Label:          ptr("foobar"),
			ExpireTime:     ptr(expire),
			Params:         nil,
			Data:           map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, p)
	c.Assert(err, jc.ErrorIsNil)

	another := s.Factory.MakeApplication(c, &factory.ApplicationParams{
		Charm: s.Factory.MakeCharm(c, &factory.CharmParams{Name: "mariadb"}),
	})
	now2 := s.Clock.Now().Round(time.Second).UTC()
	uri2 := secrets.NewURI()
	p2 := state.CreateSecretParams{
		Version: 1,
		Owner:   another.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err = s.store.CreateSecret(uri2, p2)
	c.Assert(err, jc.ErrorIsNil)

	// Create another secret to ensure it is excluded.
	uri3 := secrets.NewURI()
	p.Owner = names.NewApplicationTag("wordpress")
	_, err = s.store.CreateSecret(uri3, p)
	c.Assert(err, jc.ErrorIsNil)

	expectedList := []*secrets.SecretMetadata{{
		URI:              uri,
		RotatePolicy:     secrets.RotateDaily,
		NextRotateTime:   ptr(next),
		LatestRevision:   1,
		LatestExpireTime: ptr(expire),
		Version:          1,
		OwnerTag:         s.owner.Tag().String(),
		Description:      "my secret",
		Label:            "foobar",
		CreateTime:       now,
		UpdateTime:       now,
	}, {
		URI:            uri2,
		LatestRevision: 1,
		Version:        1,
		OwnerTag:       another.Tag().String(),
		CreateTime:     now2,
		UpdateTime:     now2,
	}}
	list, err := s.store.ListSecrets(state.SecretsFilter{
		OwnerTags: []names.Tag{s.owner.Tag(), names.NewApplicationTag("mariadb")},
	})
	c.Assert(err, jc.ErrorIsNil)

	mc := jc.NewMultiChecker()
	mc.AddExpr(`_.CreateTime`, jc.Almost, jc.ExpectedValue)
	mc.AddExpr(`_.UpdateTime`, jc.Almost, jc.ExpectedValue)

	sortMD := func(l []*secrets.SecretMetadata) {
		sort.Slice(l, func(i, j int) bool {
			return l[i].URI.String() < l[j].URI.String()
		})
	}
	sortMD(list)
	sortMD(expectedList)
	c.Assert(list, mc, expectedList)
}

func (s *SecretsSuite) TestListByURI(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	expire := now.Add(time.Hour).Round(time.Second).UTC()
	p := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Description:    ptr("my secret"),
			Label:          ptr("foobar"),
			ExpireTime:     ptr(expire),
			Params:         nil,
			Data:           map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, p)
	c.Assert(err, jc.ErrorIsNil)

	// Create another secret to ensure it is excluded.
	uri2 := secrets.NewURI()
	p.Owner = names.NewApplicationTag("wordpress")
	_, err = s.store.CreateSecret(uri2, p)
	c.Assert(err, jc.ErrorIsNil)

	list, err := s.store.ListSecrets(state.SecretsFilter{
		URI: uri,
	})
	c.Assert(err, jc.ErrorIsNil)
	mc := jc.NewMultiChecker()
	mc.AddExpr(`_.CreateTime`, jc.Almost, jc.ExpectedValue)
	mc.AddExpr(`_.UpdateTime`, jc.Almost, jc.ExpectedValue)
	c.Assert(list, mc, []*secrets.SecretMetadata{{
		URI:              uri,
		RotatePolicy:     secrets.RotateDaily,
		NextRotateTime:   ptr(next),
		LatestRevision:   1,
		LatestExpireTime: ptr(expire),
		Version:          1,
		OwnerTag:         s.owner.Tag().String(),
		Description:      "my secret",
		Label:            "foobar",
		CreateTime:       now,
		UpdateTime:       now,
	}})
}

func (s *SecretsSuite) TestListByConsumer(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	subject := names.NewApplicationTag("wordpress")

	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Description: ptr("my secret"),
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)

	err = s.State.GrantSecretAccess(uri, state.SecretAccessParams{
		LeaderToken: &fakeToken{},
		Scope:       s.relation.Tag(),
		Subject:     subject,
		Role:        secrets.RoleView,
	})
	c.Assert(err, jc.ErrorIsNil)

	// Create another secret to ensure it is excluded.
	uri2 := secrets.NewURI()
	cp.Owner = names.NewApplicationTag("wordpress")
	_, err = s.store.CreateSecret(uri2, cp)
	c.Assert(err, jc.ErrorIsNil)

	list, err := s.store.ListSecrets(state.SecretsFilter{
		ConsumerTags: []names.Tag{subject},
	})
	c.Assert(err, jc.ErrorIsNil)
	mc := jc.NewMultiChecker()
	mc.AddExpr(`_.CreateTime`, jc.Almost, jc.ExpectedValue)
	mc.AddExpr(`_.UpdateTime`, jc.Almost, jc.ExpectedValue)
	c.Assert(list, mc, []*secrets.SecretMetadata{{
		URI:            uri,
		LatestRevision: 1,
		Version:        1,
		OwnerTag:       s.owner.Tag().String(),
		Description:    "my secret",
		CreateTime:     now,
		UpdateTime:     now,
	}})
}

func (s *SecretsSuite) TestUpdateNothing(c *gc.C) {
	up := state.UpdateSecretParams{}
	uri := secrets.NewURI()
	_, err := s.store.UpdateSecret(uri, up)
	c.Assert(err, gc.ErrorMatches, "must specify a new value or metadata to update a secret")
}

func (s *SecretsSuite) TestUpdateAll(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Description:    ptr("my secret"),
			Label:          ptr("foobar"),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	newData := map[string]string{"foo": "bar", "hello": "world"}
	s.assertUpdatedSecret(c, md, 2, state.UpdateSecretParams{
		LeaderToken:    &fakeToken{},
		Description:    ptr("big secret"),
		Label:          ptr("new label"),
		RotatePolicy:   ptr(secrets.RotateHourly),
		NextRotateTime: ptr(next),
		Data:           newData,
	})
}

func (s *SecretsSuite) TestUpdateRotateInterval(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	s.assertUpdatedSecret(c, md, 1, state.UpdateSecretParams{
		LeaderToken:    &fakeToken{},
		RotatePolicy:   ptr(secrets.RotateHourly),
		NextRotateTime: ptr(next),
	})
}

func (s *SecretsSuite) TestUpdateExpiry(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	s.assertUpdatedSecret(c, md, 1, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		ExpireTime:  ptr(next),
	})

	s.assertUpdatedSecret(c, md, 1, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		ExpireTime:  ptr(time.Time{}),
	})
}

func (s *SecretsSuite) TestUpdateDuplicateLabel(c *gc.C) {
	uri := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Label:       ptr("label"),
			Description: ptr("description"),
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	uri2 := secrets.NewURI()
	cp.Label = ptr("label2")
	_, err = s.store.CreateSecret(uri2, cp)
	c.Assert(err, jc.ErrorIsNil)
	_, err = s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Label:       ptr("label2"),
	})
	c.Assert(errors.Is(err, state.LabelExists), jc.IsTrue)

	_, err = s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Label:       ptr("label"),
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *SecretsSuite) TestUpdateData(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	newData := map[string]string{"foo": "bar", "hello": "world"}
	s.assertUpdatedSecret(c, md, 2, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        newData,
	})
}

func (s *SecretsSuite) TestUpdateDataSetsLatestConsumerRevision(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	cmd := &secrets.SecretConsumerMetadata{
		Label:           "foobar",
		CurrentRevision: 1,
	}
	err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), cmd)
	c.Assert(err, jc.ErrorIsNil)
	newData := map[string]string{"foo": "bar", "hello": "world"}
	s.assertUpdatedSecret(c, md, 2, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        newData,
	})
	cmd, err = s.State.GetSecretConsumer(uri, names.NewUnitTag("mariadb/0"))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(cmd, jc.DeepEquals, &secrets.SecretConsumerMetadata{
		Label:           "foobar",
		CurrentRevision: 1,
		LatestRevision:  2,
	})
}

func (s *SecretsSuite) TestUpdateDataSetsLatestConsumerRevisionConcurrentAdd(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(now.Add(time.Minute)),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	cmd := &secrets.SecretConsumerMetadata{
		Label:           "foobar",
		CurrentRevision: 1,
	}
	err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), cmd)
	c.Assert(err, jc.ErrorIsNil)

	state.SetBeforeHooks(c, s.State, func() {
		err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), cmd)
		c.Assert(err, jc.ErrorIsNil)
	})

	newData := map[string]string{"foo": "bar", "hello": "world"}
	s.assertUpdatedSecret(c, md, 2, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        newData,
	})
	cmd, err = s.State.GetSecretConsumer(uri, names.NewUnitTag("mariadb/0"))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(cmd.LatestRevision, gc.Equals, 2)
	cmd, err = s.State.GetSecretConsumer(uri, names.NewUnitTag("mariadb/0"))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(cmd.LatestRevision, gc.Equals, 2)
}

func (s *SecretsSuite) TestUpdateDataSetsLatestConsumerRevisionConcurrentRemove(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	cmd := &secrets.SecretConsumerMetadata{
		Label:           "foobar",
		CurrentRevision: 1,
	}
	err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), cmd)
	c.Assert(err, jc.ErrorIsNil)
	err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mysql/0"), cmd)
	c.Assert(err, jc.ErrorIsNil)

	state.SetBeforeHooks(c, s.State, func() {
		consColl, closer := state.GetCollection(s.State, "secretConsumers")
		defer closer()
		err := consColl.Writeable().RemoveId(state.DocID(s.State, fmt.Sprintf("%s#unit-mysql-0", uri.ID)))
		c.Assert(err, jc.ErrorIsNil)

		err = state.IncSecretConsumerRefCount(s.State, uri, 1)
		c.Assert(err, jc.ErrorIsNil)
	})

	newData := map[string]string{"foo": "bar", "hello": "world"}
	s.assertUpdatedSecret(c, md, 2, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        newData,
	})
	cmd, err = s.State.GetSecretConsumer(uri, names.NewUnitTag("mariadb/0"))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(cmd.LatestRevision, gc.Equals, 2)
	_, err = s.State.GetSecretConsumer(uri, names.NewUnitTag("mysql/0"))
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
}

func (s *SecretsSuite) assertUpdatedSecret(c *gc.C, original *secrets.SecretMetadata, expectedRevision int, update state.UpdateSecretParams) {
	expected := *original
	expected.LatestRevision = expectedRevision
	if update.RotatePolicy != nil {
		expected.RotatePolicy = *update.RotatePolicy
		expected.NextRotateTime = update.NextRotateTime
	}
	if update.Description != nil {
		expected.Description = *update.Description
	}
	if update.Label != nil {
		expected.Label = *update.Label
	}
	if update.ExpireTime != nil && !update.ExpireTime.IsZero() {
		expected.LatestExpireTime = update.ExpireTime
	}

	s.Clock.Advance(time.Hour)
	updated := s.Clock.Now().Round(time.Second).UTC()
	expected.UpdateTime = updated
	md, err := s.store.UpdateSecret(original.URI, update)
	c.Assert(err, jc.ErrorIsNil)

	list, err := s.store.ListSecrets(state.SecretsFilter{})
	c.Assert(err, jc.ErrorIsNil)
	mc := jc.NewMultiChecker()
	mc.AddExpr(`(*_[_]).CreateTime`, jc.Almost, jc.ExpectedValue)
	mc.AddExpr(`(*_[_]).UpdateTime`, jc.Almost, jc.ExpectedValue)
	c.Assert(list, mc, []*secrets.SecretMetadata{&expected})
	expectedData := map[string]string{"foo": "bar"}
	if update.Data != nil {
		expectedData = update.Data
	}
	val, providerId, err := s.store.GetSecretValue(md.URI, expectedRevision)
	c.Assert(err, jc.ErrorIsNil)
	if update.ProviderId != nil {
		c.Assert(providerId, gc.NotNil)
		c.Assert(*update.ProviderId, gc.Equals, *providerId)
	} else {
		c.Assert(providerId, gc.IsNil)
		c.Assert(val.EncodedValues(), jc.DeepEquals, expectedData)
	}
	if update.Label != nil {
		uri, err := s.store.GetURIBySecretLabel(*update.Label, s.owner.Tag())
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(uri, gc.DeepEquals, original.URI)
	}
	if update.ExpireTime != nil {
		revs, err := s.store.ListSecretRevisions(md.URI)
		c.Assert(err, jc.ErrorIsNil)
		for _, r := range revs {
			if r.ExpireTime == nil && update.ExpireTime.IsZero() {
				return
			}
			if r.ExpireTime != nil && r.ExpireTime.Equal(update.ExpireTime.Round(time.Second).UTC()) {
				return
			}
		}
		c.Fatalf("expire time not set for secret revision %d", expectedRevision)
		md, err := s.store.GetSecret(original.URI)
		c.Assert(err, jc.ErrorIsNil)
		if update.ExpireTime.IsZero() {
			c.Assert(md.LatestExpireTime, gc.IsNil)
		} else {
			c.Assert(md.LatestExpireTime, gc.Equals, update.ExpireTime.Round(time.Second).UTC())
		}
	}
	if update.NextRotateTime != nil {
		nextTime := state.GetSecretNextRotateTime(c, s.State, md.URI.ID)
		c.Assert(nextTime, gc.Equals, *update.NextRotateTime)
	}
}

func (s *SecretsSuite) TestUpdateConcurrent(c *gc.C) {
	uri := secrets.NewURI()

	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)

	state.SetBeforeHooks(c, s.State, func() {
		up := state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateYearly),
			NextRotateTime: ptr(next),
			Params:         nil,
			Data:           map[string]string{"foo": "baz", "goodbye": "world"},
		}
		md, err = s.store.UpdateSecret(md.URI, up)
		c.Assert(err, jc.ErrorIsNil)
	})
	newData := map[string]string{"foo": "bar", "hello": "world"}
	s.assertUpdatedSecret(c, md, 3, state.UpdateSecretParams{
		LeaderToken:    &fakeToken{},
		RotatePolicy:   ptr(secrets.RotateHourly),
		NextRotateTime: ptr(next),
		Data:           newData,
	})
}

func (s *SecretsSuite) TestGetSecretAndGetSecretURI(c *gc.C) {
	uri := secrets.NewURI()

	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			Label:          strPtr("label-1"),
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Check(err, jc.ErrorIsNil)
	c.Check(md.URI, jc.DeepEquals, uri)

	md, err = s.store.GetSecret(uri)
	c.Check(err, jc.ErrorIsNil)
	c.Check(md.URI, jc.DeepEquals, uri)

	result, err := s.store.GetURIBySecretLabel("label-1", s.owner.Tag())
	c.Check(err, jc.ErrorIsNil)
	c.Check(result, jc.DeepEquals, uri)
}

func (s *SecretsSuite) TestListSecretRevisions(c *gc.C) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	newData := map[string]string{"foo": "bar", "hello": "world"}
	s.assertUpdatedSecret(c, md, 2, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        newData,
	})
	updateTime := s.Clock.Now().Round(time.Second).UTC()
	s.assertUpdatedSecret(c, md, 3, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		ProviderId:  ptr("provider-id"),
	})
	updateTime2 := s.Clock.Now().Round(time.Second).UTC()
	r, err := s.store.ListSecretRevisions(uri)
	c.Assert(err, jc.ErrorIsNil)

	mc := jc.NewMultiChecker()
	mc.AddExpr(`_.CreateTime`, jc.Almost, jc.ExpectedValue)
	mc.AddExpr(`_.UpdateTime`, jc.Almost, jc.ExpectedValue)
	c.Assert(r, mc, []*secrets.SecretRevisionMetadata{{
		Revision:   1,
		CreateTime: now,
		UpdateTime: now,
	}, {
		Revision:   2,
		CreateTime: updateTime,
		UpdateTime: updateTime,
	}, {
		Revision:   3,
		ProviderId: ptr("provider-id"),
		CreateTime: updateTime2,
		UpdateTime: updateTime2,
	}})
}

func (s *SecretsSuite) TestGetSecretRevision(c *gc.C) {
	uri := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	newData := map[string]string{"foo": "bar", "hello": "world"}
	s.assertUpdatedSecret(c, md, 2, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        newData,
	})
	r, err := s.store.GetSecretRevision(uri, 2)
	c.Assert(err, jc.ErrorIsNil)
	updateTime := s.Clock.Now().Round(time.Second).UTC()
	mc := jc.NewMultiChecker()
	mc.AddExpr(`_.CreateTime`, jc.Almost, jc.ExpectedValue)
	mc.AddExpr(`_.UpdateTime`, jc.Almost, jc.ExpectedValue)
	c.Assert(r, mc, &secrets.SecretRevisionMetadata{
		Revision:   2,
		CreateTime: updateTime,
		UpdateTime: updateTime,
	})
}

func (s *SecretsSuite) TestGetSecretConsumerAndGetSecretConsumerURI(c *gc.C) {
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	uri := secrets.NewURI()
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)

	_, err = s.State.GetSecretConsumer(uri, names.NewUnitTag("mariadb/0"))
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
	md := &secrets.SecretConsumerMetadata{
		Label:           "foobar",
		CurrentRevision: 666,
	}
	err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), md)
	c.Assert(err, jc.ErrorIsNil)

	_, err = s.State.GetSecretConsumer(nil, names.NewUnitTag("mariadb/0"))
	c.Check(err, gc.ErrorMatches, `empty URI`)

	md2, err := s.State.GetSecretConsumer(uri, names.NewUnitTag("mariadb/0"))
	c.Check(err, jc.ErrorIsNil)
	c.Check(md2, jc.DeepEquals, md)

	uri3, err := s.State.GetURIByConsumerLabel("foobar", names.NewUnitTag("mariadb/0"))
	c.Check(err, jc.ErrorIsNil)
	c.Check(uri3, jc.DeepEquals, uri)

	_, err = s.State.GetSecretConsumer(uri, names.NewUnitTag("mysql/0"))
	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

func (s *SecretsSuite) TestSaveSecretConsumer(c *gc.C) {
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	uri := secrets.NewURI()
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	md := &secrets.SecretConsumerMetadata{
		Label:           "foobar",
		CurrentRevision: 666,
	}
	err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), md)
	c.Assert(err, jc.ErrorIsNil)
	md2, err := s.State.GetSecretConsumer(uri, names.NewUnitTag("mariadb/0"))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(md2, jc.DeepEquals, md)
	md.CurrentRevision = 668
	err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), md)
	c.Assert(err, jc.ErrorIsNil)
	md2, err = s.State.GetSecretConsumer(uri, names.NewUnitTag("mariadb/0"))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(md2, jc.DeepEquals, md)
}

func (s *SecretsSuite) TestSaveSecretConsumerConcurrent(c *gc.C) {
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	uri := secrets.NewURI()
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	md := &secrets.SecretConsumerMetadata{
		Label:           "foobar",
		CurrentRevision: 666,
	}
	state.SetBeforeHooks(c, s.State, func() {
		err := s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), &secrets.SecretConsumerMetadata{CurrentRevision: 668})
		c.Assert(err, jc.ErrorIsNil)
	})
	err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), md)
	c.Assert(err, jc.ErrorIsNil)
	md2, err := s.State.GetSecretConsumer(uri, names.NewUnitTag("mariadb/0"))
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(md2, jc.DeepEquals, md)
}

func (s *SecretsSuite) TestSecretGrantAccess(c *gc.C) {
	uri := secrets.NewURI()
	subject := names.NewApplicationTag("wordpress")
	err := s.State.GrantSecretAccess(uri, state.SecretAccessParams{
		LeaderToken: &fakeToken{},
		Scope:       s.relation.Tag(),
		Subject:     subject,
		Role:        secrets.RoleView,
	})
	c.Assert(err, jc.Satisfies, errors.IsNotFound)

	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err = s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)

	err = s.State.GrantSecretAccess(uri, state.SecretAccessParams{
		LeaderToken: &fakeToken{},
		Scope:       s.relation.Tag(),
		Subject:     subject,
		Role:        secrets.RoleView,
	})
	c.Assert(err, jc.ErrorIsNil)
	access, err := s.State.SecretAccess(uri, subject)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(access, gc.Equals, secrets.RoleView)
}

func (s *SecretsSuite) TestSecretGrantCrossModel(c *gc.C) {
	rwordpress, err := s.State.AddRemoteApplication(state.AddRemoteApplicationParams{
		Name:            "remote-wordpress",
		SourceModel:     names.NewModelTag("source-model"),
		IsConsumerProxy: true,
		OfferUUID:       "offer-uuid",
		Endpoints: []charm.Relation{{
			Interface: "mysql",
			Limit:     1,
			Name:      "db",
			Role:      charm.RoleRequirer,
			Scope:     charm.ScopeGlobal,
		}},
	})
	c.Assert(err, jc.ErrorIsNil)
	wordpressEP, err := rwordpress.Endpoint("db")
	c.Assert(err, jc.ErrorIsNil)
	mysqlEP, err := s.owner.Endpoint("server")
	c.Assert(err, jc.ErrorIsNil)
	relation, err := s.State.AddRelation(wordpressEP, mysqlEP)
	c.Assert(err, jc.ErrorIsNil)

	uri := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err = s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)

	err = s.State.GrantSecretAccess(uri, state.SecretAccessParams{
		LeaderToken: &fakeToken{},
		Scope:       relation.Tag(),
		Subject:     rwordpress.Tag(),
		Role:        secrets.RoleView,
	})
	c.Assert(err, jc.Satisfies, errors.IsNotSupported)
}

func (s *SecretsSuite) TestSecretGrantAccessDyingScope(c *gc.C) {
	uri := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)

	// Ensure destroy only sets relation to dying.
	wordpress, err := s.State.Application("wordpress")
	c.Assert(err, jc.ErrorIsNil)
	unit, err := wordpress.AddUnit(state.AddUnitParams{})
	c.Assert(err, jc.ErrorIsNil)
	ru, err := s.relation.Unit(unit)
	c.Assert(err, jc.ErrorIsNil)
	err = ru.EnterScope(nil)
	c.Assert(err, jc.ErrorIsNil)

	_, err = s.relation.DestroyWithForce(true, time.Second)
	c.Assert(err, jc.ErrorIsNil)

	err = s.State.GrantSecretAccess(uri, state.SecretAccessParams{
		LeaderToken: &fakeToken{},
		Scope:       s.relation.Tag(),
		Subject:     wordpress.Tag(),
		Role:        secrets.RoleView,
	})
	c.Assert(err, gc.ErrorMatches, `cannot grant access to secret in scope of "relation-wordpress.db#mysql.server" which is not alive`)
}

func (s *SecretsSuite) TestSecretGrantAccessDyingSubject(c *gc.C) {
	uri := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)

	// Ensure destroy only sets app to dying.
	wordpress, err := s.State.Application("wordpress")
	c.Assert(err, jc.ErrorIsNil)
	unit, err := wordpress.AddUnit(state.AddUnitParams{})
	c.Assert(err, jc.ErrorIsNil)
	ru, err := s.relation.Unit(unit)
	c.Assert(err, jc.ErrorIsNil)
	err = ru.EnterScope(nil)
	c.Assert(err, jc.ErrorIsNil)

	err = wordpress.Destroy()
	c.Assert(err, jc.ErrorIsNil)

	err = s.State.GrantSecretAccess(uri, state.SecretAccessParams{
		LeaderToken: &fakeToken{},
		Scope:       s.relation.Tag(),
		Subject:     wordpress.Tag(),
		Role:        secrets.RoleView,
	})
	c.Assert(err, gc.ErrorMatches, `cannot grant access to secret in scope of "relation-wordpress.db#mysql.server" which is not alive`)
}

func (s *SecretsSuite) TestSecretRevokeAccess(c *gc.C) {
	uri := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)

	subject := names.NewApplicationTag("wordpress")
	err = s.State.GrantSecretAccess(uri, state.SecretAccessParams{
		LeaderToken: &fakeToken{},
		Scope:       s.relation.Tag(),
		Subject:     subject,
		Role:        secrets.RoleView,
	})
	c.Assert(err, jc.ErrorIsNil)
	access, err := s.State.SecretAccess(uri, subject)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(access, gc.Equals, secrets.RoleView)

	err = s.State.RevokeSecretAccess(uri, state.SecretAccessParams{
		LeaderToken: &fakeToken{},
		Subject:     subject,
	})
	c.Assert(err, jc.ErrorIsNil)
	access, err = s.State.SecretAccess(uri, subject)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(access, gc.Equals, secrets.RoleNone)

	err = s.State.RevokeSecretAccess(uri, state.SecretAccessParams{
		LeaderToken: &fakeToken{},
		Subject:     subject,
	})
	c.Assert(err, jc.ErrorIsNil)
}

func (s *SecretsSuite) TestDelete(c *gc.C) {
	subject := names.NewApplicationTag("wordpress")
	create := func(label string) *secrets.URI {
		uri := secrets.NewURI()
		now := s.Clock.Now().Round(time.Second).UTC()
		next := now.Add(time.Minute).Round(time.Second).UTC()
		cp := state.CreateSecretParams{
			Version: 1,
			Owner:   s.owner.Tag(),
			UpdateSecretParams: state.UpdateSecretParams{
				LeaderToken:    &fakeToken{},
				RotatePolicy:   ptr(secrets.RotateDaily),
				NextRotateTime: ptr(next),
				Label:          ptr(label),
				Data:           map[string]string{"foo": "bar"},
			},
		}
		_, err := s.store.CreateSecret(uri, cp)
		c.Assert(err, jc.ErrorIsNil)
		cmd := &secrets.SecretConsumerMetadata{
			Label:           "consumer-" + label,
			CurrentRevision: 1,
		}
		err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), cmd)
		c.Assert(err, jc.ErrorIsNil)
		err = s.State.GrantSecretAccess(uri, state.SecretAccessParams{
			LeaderToken: &fakeToken{},
			Scope:       s.relation.Tag(),
			Subject:     subject,
			Role:        secrets.RoleView,
		})
		c.Assert(err, jc.ErrorIsNil)
		return uri
	}
	uri1 := create("label1")
	uri2 := create("label2")

	removed, err := s.store.DeleteSecret(uri1)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(removed, jc.IsTrue)
	_, _, err = s.store.GetSecretValue(uri1, 1)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
	removed, err = s.store.DeleteSecret(uri1)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(removed, jc.IsTrue)

	// Check that other secret info remains intact.
	secretRevisionsCollection, closer := state.GetCollection(s.State, "secretRevisions")
	defer closer()
	n, err := secretRevisionsCollection.FindId(uri2.ID + "/1").Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(n, gc.Equals, 1)
	n, err = secretRevisionsCollection.Find(nil).Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(n, gc.Equals, 1)

	secretRotateCollection, closer := state.GetCollection(s.State, "secretRotate")
	defer closer()
	n, err = secretRotateCollection.FindId(uri2.ID).Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(n, gc.Equals, 1)
	n, err = secretRotateCollection.Find(nil).Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(n, gc.Equals, 1)

	secretConsumersCollection, closer := state.GetCollection(s.State, "secretConsumers")
	defer closer()
	n, err = secretConsumersCollection.FindId(uri2.ID + "#unit-mariadb-0").Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(n, gc.Equals, 1)
	n, err = secretConsumersCollection.Find(nil).Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(n, gc.Equals, 1)

	secretPermissionsCollection, closer := state.GetCollection(s.State, "secretPermissions")
	defer closer()
	n, err = secretPermissionsCollection.FindId(uri2.ID + "#application-wordpress").Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(n, gc.Equals, 1)
	n, err = secretPermissionsCollection.Find(nil).Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(n, gc.Equals, 1)

	refCountsCollection, closer := state.GetCollection(s.State, "refcounts")
	defer closer()
	n, err = refCountsCollection.FindId(uri2.ID + "#consumer").Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(n, gc.Equals, 1)
	n, err = refCountsCollection.FindId(uri1.ID + "#consumer").Count()
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(n, gc.Equals, 0)

	// Check we can now reuse the label.
	create("label1")
}

func (s *SecretsSuite) TestDeleteRevisions(c *gc.C) {
	uri := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	_, err = s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        map[string]string{"foo": "bar2"},
	})
	c.Assert(err, jc.ErrorIsNil)
	_, err = s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        map[string]string{"foo": "bar3"},
	})
	c.Assert(err, jc.ErrorIsNil)

	removed, err := s.store.DeleteSecret(uri, 1)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(removed, jc.IsFalse)
	_, _, err = s.store.GetSecretValue(uri, 1)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
	val, _, err := s.store.GetSecretValue(uri, 2)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(val.EncodedValues(), jc.DeepEquals, map[string]string{"foo": "bar2"})
	val, _, err = s.store.GetSecretValue(uri, 3)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(val.EncodedValues(), jc.DeepEquals, map[string]string{"foo": "bar3"})

	removed, err = s.store.DeleteSecret(uri, 1, 2, 3)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(removed, jc.IsTrue)
	_, _, err = s.store.GetSecretValue(uri, 3)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
	_, err = s.store.GetSecret(uri)
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
}

func (s *SecretsSuite) TestSecretRotated(c *gc.C) {
	uri := secrets.NewURI()

	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	next2 := now.Add(time.Hour).Round(time.Second).UTC()
	err = s.State.SecretRotated(uri, next2)
	c.Assert(err, jc.ErrorIsNil)

	nextTime := state.GetSecretNextRotateTime(c, s.State, md.URI.ID)
	c.Assert(nextTime, gc.Equals, next2)
}

func (s *SecretsSuite) TestSecretRotatedConcurrent(c *gc.C) {
	uri := secrets.NewURI()

	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)

	later := now.Add(time.Hour).Round(time.Second).UTC()
	later2 := now.Add(2 * time.Hour).Round(time.Second).UTC()
	state.SetBeforeHooks(c, s.State, func() {
		err := s.State.SecretRotated(uri, later)
		c.Assert(err, jc.ErrorIsNil)
	})

	err = s.State.SecretRotated(uri, later2)
	c.Assert(err, jc.ErrorIsNil)

	nextTime := state.GetSecretNextRotateTime(c, s.State, md.URI.ID)
	c.Assert(nextTime, gc.Equals, later)
}

type SecretsRotationWatcherSuite struct {
	testing.StateSuite
	store state.SecretsStore

	ownerApp  *state.Application
	ownerUnit *state.Unit
}

var _ = gc.Suite(&SecretsRotationWatcherSuite{})

func (s *SecretsRotationWatcherSuite) SetUpTest(c *gc.C) {
	s.StateSuite.SetUpTest(c)
	s.store = state.NewSecrets(s.State)
	s.ownerApp = s.Factory.MakeApplication(c, nil)
	s.ownerUnit = s.Factory.MakeUnit(c, &factory.UnitParams{Application: s.ownerApp})
}

func (s *SecretsRotationWatcherSuite) setupWatcher(c *gc.C) (state.SecretsTriggerWatcher, *secrets.URI) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.ownerApp.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateDaily),
			NextRotateTime: ptr(next),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	w, err := s.State.WatchSecretsRotationChanges(
		[]names.Tag{s.ownerApp.Tag(), s.ownerUnit.Tag()})
	c.Assert(err, jc.ErrorIsNil)

	wc := testing.NewSecretsTriggerWatcherC(c, w)
	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             md.URI,
		NextTriggerTime: next,
	})
	wc.AssertNoChange()
	return w, uri
}

func (s *SecretsRotationWatcherSuite) TestWatchInitialEvent(c *gc.C) {
	w, _ := s.setupWatcher(c)
	testing.AssertStop(c, w)
}

func (s *SecretsRotationWatcherSuite) TestWatchSingleUpdate(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(2 * time.Hour).Round(time.Second).UTC()
	err := s.State.SecretRotated(uri, next)
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri,
		NextTriggerTime: next,
	})
	wc.AssertNoChange()
}

func (s *SecretsRotationWatcherSuite) TestWatchDelete(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	md, err := s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken:  &fakeToken{},
		RotatePolicy: ptr(secrets.RotateNever),
	})
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI: md.URI,
	})
	wc.AssertNoChange()
}

func (s *SecretsRotationWatcherSuite) TestWatchMultipleUpdatesSameSecret(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	// TODO(quiescence): these two changes should be one event.
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	err := s.State.SecretRotated(uri, next)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri,
		NextTriggerTime: next,
	})
	next2 := now.Add(time.Hour).Round(time.Second).UTC()
	err = s.State.SecretRotated(uri, next2)
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri,
		NextTriggerTime: next2,
	})
	wc.AssertNoChange()
}

func (s *SecretsRotationWatcherSuite) TestWatchMultipleUpdatesSameSecretDeleted(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	// TODO(quiescence): these two changes should be one event.
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Hour).Round(time.Second).UTC()
	err := s.State.SecretRotated(uri, next)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri,
		NextTriggerTime: next,
	})
	md, err := s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken:  &fakeToken{},
		RotatePolicy: ptr(secrets.RotateNever),
	})
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI: md.URI,
	})
	wc.AssertNoChange()
}

func (s *SecretsRotationWatcherSuite) TestWatchMultipleUpdates(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	// TODO(quiescence): these two changes should be one event.
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Hour).Round(time.Second).UTC()
	err := s.State.SecretRotated(uri, next)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri,
		NextTriggerTime: next,
	})

	uri2 := secrets.NewURI()
	next2 := now.Add(time.Minute).Round(time.Second).UTC()
	md2, err := s.store.CreateSecret(uri2, state.CreateSecretParams{
		Version: 1,
		Owner:   s.ownerApp.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateHourly),
			NextRotateTime: ptr(next2),
			Data:           map[string]string{"foo": "bar"},
		},
	})
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             md2.URI,
		NextTriggerTime: next2,
	})

	md, err := s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken:  &fakeToken{},
		RotatePolicy: ptr(secrets.RotateNever),
	})
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI: md.URI,
	})
	wc.AssertNoChange()
}

func (s *SecretsRotationWatcherSuite) TestWatchRestartChangeOwners(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	now := s.Clock.Now().Round(time.Second).UTC()
	next1 := now.Add(time.Minute).Round(time.Second).UTC()
	next2 := now.Add(time.Minute).Round(time.Second).UTC()

	uri2 := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.ownerUnit.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateHourly),
			NextRotateTime: ptr(next2),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri2, cp)
	c.Assert(err, jc.ErrorIsNil)

	next3 := now.Add(time.Minute).Round(time.Second).UTC()
	anotherUnit := s.Factory.MakeUnit(c, &factory.UnitParams{Application: s.ownerApp})

	uri3 := secrets.NewURI()
	cp = state.CreateSecretParams{
		Version: 1,
		Owner:   anotherUnit.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken:    &fakeToken{},
			RotatePolicy:   ptr(secrets.RotateHourly),
			NextRotateTime: ptr(next3),
			Data:           map[string]string{"foo": "bar"},
		},
	}
	_, err = s.store.CreateSecret(uri3, cp)
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri2,
		NextTriggerTime: next2,
	})

	wc.AssertNoChange()
	testing.AssertStop(c, w)

	w, err = s.State.WatchSecretsRotationChanges(
		[]names.Tag{s.ownerApp.Tag(), anotherUnit.Tag()})
	c.Assert(err, jc.ErrorIsNil)

	wc = testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri,
		NextTriggerTime: next1,
	}, watcher.SecretTriggerChange{
		URI:             uri3,
		NextTriggerTime: next3,
	})
	wc.AssertNoChange()
}

type SecretsExpiryWatcherSuite struct {
	testing.StateSuite
	store state.SecretsStore

	ownerApp  *state.Application
	ownerUnit *state.Unit
}

var _ = gc.Suite(&SecretsExpiryWatcherSuite{})

func (s *SecretsExpiryWatcherSuite) SetUpTest(c *gc.C) {
	s.StateSuite.SetUpTest(c)
	s.store = state.NewSecrets(s.State)
	s.ownerApp = s.Factory.MakeApplication(c, nil)
	s.ownerUnit = s.Factory.MakeUnit(c, &factory.UnitParams{Application: s.ownerApp})
}

func (s *SecretsExpiryWatcherSuite) setupWatcher(c *gc.C) (state.SecretsTriggerWatcher, *secrets.URI) {
	uri := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.ownerApp.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			ExpireTime:  ptr(next),
			Data:        map[string]string{"foo": "bar"},
		},
	}
	md, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	w, err := s.State.WatchSecretRevisionsExpiryChanges(
		[]names.Tag{s.ownerApp.Tag(), s.ownerUnit.Tag()})
	c.Assert(err, jc.ErrorIsNil)

	wc := testing.NewSecretsTriggerWatcherC(c, w)
	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             md.URI,
		Revision:        1,
		NextTriggerTime: next,
	})
	wc.AssertNoChange()
	return w, uri
}

func (s *SecretsExpiryWatcherSuite) TestWatchInitialEvent(c *gc.C) {
	w, _ := s.setupWatcher(c)
	testing.AssertStop(c, w)
}

func (s *SecretsExpiryWatcherSuite) TestWatchSingleUpdate(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(2 * time.Hour).Round(time.Second).UTC()

	s.Clock.Advance(time.Hour)
	updated := s.Clock.Now().Round(time.Second).UTC()
	update := state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		ExpireTime:  ptr(next),
	}
	md, err := s.store.UpdateSecret(uri, update)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(md.LatestExpireTime, gc.NotNil)
	c.Assert(*md.LatestExpireTime, gc.Equals, next)

	revs, err := s.store.ListSecretRevisions(md.URI)
	c.Assert(err, jc.ErrorIsNil)
	for _, r := range revs {
		if r.ExpireTime != nil && r.ExpireTime.Equal(update.ExpireTime.Round(time.Second).UTC()) {
			c.Assert(r.UpdateTime, jc.Almost, updated)
			return
		}
	}
	c.Fatalf("expire time not set for secret revision %d", 2)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri,
		Revision:        3,
		NextTriggerTime: next,
	})
	wc.AssertNoChange()
}

func (s *SecretsExpiryWatcherSuite) TestWatchSetExpiryToNil(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	md, err := s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		ExpireTime:  ptr(time.Time{}),
	})
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:      md.URI,
		Revision: 1,
	})
	wc.AssertNoChange()
}

func (s *SecretsExpiryWatcherSuite) TestWatchMultipleUpdates(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	now := s.Clock.Now().Round(time.Second).UTC()
	md, err := s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		ExpireTime:  ptr(time.Time{}),
	})
	c.Assert(err, jc.ErrorIsNil)

	next := now.Add(2 * time.Hour).Round(time.Second).UTC()
	update := state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		ExpireTime:  ptr(next),
	}
	_, err = s.store.UpdateSecret(uri, update)
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:      md.URI,
		Revision: 1,
	}, watcher.SecretTriggerChange{
		URI:             md.URI,
		Revision:        1,
		NextTriggerTime: next,
	})
	wc.AssertNoChange()
}

func (s *SecretsExpiryWatcherSuite) TestWatchRemoveSecret(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	_, err := s.store.DeleteSecret(uri)
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:      uri,
		Revision: 1,
	})
	wc.AssertNoChange()

	uri2 := secrets.NewURI()
	now := s.Clock.Now().Round(time.Second).UTC()
	next := now.Add(time.Minute).Round(time.Second).UTC()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.ownerUnit.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			ExpireTime:  ptr(next),
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err = s.store.CreateSecret(uri2, cp)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri2,
		Revision:        1,
		NextTriggerTime: next,
	})
	wc.AssertNoChange()

	_, err = s.store.DeleteSecret(uri2)
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:      uri2,
		Revision: 1,
	})
	wc.AssertNoChange()
}

func (s *SecretsExpiryWatcherSuite) TestWatchRemoveRevision(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	now := s.Clock.Now().Round(time.Second).UTC()
	triggerTime := now.Add(time.Minute).Round(time.Second).UTC()
	_, err := s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        map[string]string{"foo": "bar2"},
	})
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri,
		Revision:        1,
		NextTriggerTime: triggerTime,
	})
	wc.AssertNoChange()

	_, err = s.store.DeleteSecret(uri, 1)
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:      uri,
		Revision: 1,
	})
	wc.AssertNoChange()
}

func (s *SecretsExpiryWatcherSuite) TestWatchRestartChangeOwners(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	now := s.Clock.Now().Round(time.Second).UTC()
	next1 := now.Add(time.Minute).Round(time.Second).UTC()
	next2 := now.Add(time.Minute).Round(time.Second).UTC()

	uri2 := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.ownerUnit.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			ExpireTime:  ptr(next2),
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri2, cp)
	c.Assert(err, jc.ErrorIsNil)

	next3 := now.Add(time.Minute).Round(time.Second).UTC()

	anotherUnit := s.Factory.MakeUnit(c, &factory.UnitParams{Application: s.ownerApp})
	uri3 := secrets.NewURI()
	cp = state.CreateSecretParams{
		Version: 1,
		Owner:   anotherUnit.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			ExpireTime:  ptr(next3),
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err = s.store.CreateSecret(uri3, cp)
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri2,
		Revision:        1,
		NextTriggerTime: next2,
	})

	wc.AssertNoChange()
	testing.AssertStop(c, w)

	w, err = s.State.WatchSecretRevisionsExpiryChanges(
		[]names.Tag{s.ownerApp.Tag(), anotherUnit.Tag()})
	c.Assert(err, jc.ErrorIsNil)

	wc = testing.NewSecretsTriggerWatcherC(c, w)
	defer testing.AssertStop(c, w)

	wc.AssertChange(watcher.SecretTriggerChange{
		URI:             uri,
		Revision:        1,
		NextTriggerTime: next1,
	}, watcher.SecretTriggerChange{
		URI:             uri3,
		Revision:        1,
		NextTriggerTime: next3,
	})
	wc.AssertNoChange()
}

type SecretsConsumedWatcherSuite struct {
	testing.StateSuite
	store state.SecretsStore

	owner *state.Application
}

var _ = gc.Suite(&SecretsConsumedWatcherSuite{})

func (s *SecretsConsumedWatcherSuite) SetUpTest(c *gc.C) {
	s.StateSuite.SetUpTest(c)
	s.store = state.NewSecrets(s.State)
	s.owner = s.Factory.MakeApplication(c, nil)
}

func (s *SecretsConsumedWatcherSuite) setupWatcher(c *gc.C) (state.StringsWatcher, *secrets.URI) {
	uri := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	w, err := s.State.WatchConsumedSecretsChanges(names.NewUnitTag("mariadb/0"))
	c.Assert(err, jc.ErrorIsNil)

	wc := testing.NewStringsWatcherC(c, w)
	wc.AssertChange()

	err = s.State.SaveSecretConsumer(uri, names.NewUnitTag("mariadb/0"), &secrets.SecretConsumerMetadata{CurrentRevision: 1})
	c.Assert(err, jc.ErrorIsNil)
	// No event until rev > 1.
	wc.AssertNoChange()
	return w, uri
}

func (s *SecretsConsumedWatcherSuite) TestWatcherStartStop(c *gc.C) {
	w, _ := s.setupWatcher(c)
	testing.AssertStop(c, w)
}

func (s *SecretsConsumedWatcherSuite) TestWatchSingleUpdate(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewStringsWatcherC(c, w)
	defer testing.AssertStop(c, w)

	_, err := s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        secrets.SecretData{"foo": "bar2"},
	})
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(uri.String())
	wc.AssertNoChange()
}

func (s *SecretsConsumedWatcherSuite) TestWatchMultipleSecrets(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewStringsWatcherC(c, w)
	defer testing.AssertStop(c, w)

	uri2 := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.owner.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo2": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri2, cp)
	c.Assert(err, jc.ErrorIsNil)

	err = s.State.SaveSecretConsumer(uri2, names.NewUnitTag("mariadb/0"), &secrets.SecretConsumerMetadata{CurrentRevision: 1})
	c.Assert(err, jc.ErrorIsNil)
	// No event until rev > 1.
	wc.AssertNoChange()

	_, err = s.store.UpdateSecret(uri, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        secrets.SecretData{"foo": "bar2"},
	})
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(uri.String())
	wc.AssertNoChange()

	_, err = s.store.UpdateSecret(uri2, state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        secrets.SecretData{"foo2": "bar2"},
	})
	c.Assert(err, jc.ErrorIsNil)

	wc.AssertChange(uri2.String())
	wc.AssertNoChange()
}

func (s *SecretsConsumedWatcherSuite) TestWatchConsumedDeleted(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewStringsWatcherC(c, w)
	defer testing.AssertStop(c, w)

	err := s.State.SaveSecretConsumer(uri, names.NewApplicationTag("foo"), &secrets.SecretConsumerMetadata{
		CurrentRevision: 1,
	})
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertNoChange()
	err = s.State.SaveSecretConsumer(uri, names.NewApplicationTag("baz"), &secrets.SecretConsumerMetadata{
		CurrentRevision: 1,
	})
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertNoChange()

	_, err = s.store.DeleteSecret(uri)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(uri.String())
	wc.AssertNoChange()
}

type SecretsObsoleteWatcherSuite struct {
	testing.StateSuite
	store state.SecretsStore

	ownerApp  *state.Application
	ownerUnit *state.Unit
}

var _ = gc.Suite(&SecretsObsoleteWatcherSuite{})

func (s *SecretsObsoleteWatcherSuite) SetUpTest(c *gc.C) {
	s.StateSuite.SetUpTest(c)
	s.store = state.NewSecrets(s.State)
	s.ownerApp = s.Factory.MakeApplication(c, nil)
	s.ownerUnit = s.Factory.MakeUnit(c, &factory.UnitParams{Application: s.ownerApp})
}

func (s *SecretsObsoleteWatcherSuite) setupWatcher(c *gc.C) (state.StringsWatcher, *secrets.URI) {
	uri := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   s.ownerApp.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri, cp)
	c.Assert(err, jc.ErrorIsNil)
	w, err := s.store.WatchObsolete(
		[]names.Tag{s.ownerApp.Tag(), s.ownerUnit.Tag()})
	c.Assert(err, jc.ErrorIsNil)

	wc := testing.NewStringsWatcherC(c, w)
	wc.AssertChange()
	wc.AssertNoChange()
	return w, uri
}

func (s *SecretsObsoleteWatcherSuite) TestWatcherStartStop(c *gc.C) {
	w, _ := s.setupWatcher(c)
	testing.AssertStop(c, w)
}

func (s *SecretsObsoleteWatcherSuite) TestWatchObsoleteRevisions(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewStringsWatcherC(c, w)
	defer testing.AssertStop(c, w)

	err := s.State.SaveSecretConsumer(uri, names.NewApplicationTag("foo"), &secrets.SecretConsumerMetadata{
		CurrentRevision: 1,
	})
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertNoChange()

	p := state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        map[string]string{"foo": "bar2"},
	}
	_, err = s.store.UpdateSecret(uri, p)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertNoChange()

	err = s.State.SaveSecretConsumer(uri, names.NewApplicationTag("foo2"), &secrets.SecretConsumerMetadata{
		CurrentRevision: 2,
	})
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertNoChange()

	// The previous consumer of rev 1 now uses rev 2; rev 1 is orphaned.
	err = s.State.SaveSecretConsumer(uri, names.NewApplicationTag("foo"), &secrets.SecretConsumerMetadata{
		CurrentRevision: 2,
	})
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(uri.String() + "/1")
	wc.AssertNoChange()

	// The latest added revision is never obsolete.
	p = state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        map[string]string{"foo": "bar3"},
	}
	_, err = s.store.UpdateSecret(uri, p)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(uri.String() + "/1")
	wc.AssertNoChange()

	// New revision 4 added, so rev 3 is now also obsolete.
	p = state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        map[string]string{"foo": "bar4"},
	}
	_, err = s.store.UpdateSecret(uri, p)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(uri.String()+"/1", uri.String()+"/3")
	wc.AssertNoChange()
}

func (s *SecretsObsoleteWatcherSuite) TestWatchOwnedDeleted(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewStringsWatcherC(c, w)
	defer testing.AssertStop(c, w)

	owner2 := s.Factory.MakeApplication(c, &factory.ApplicationParams{
		Charm: s.Factory.MakeCharm(c, &factory.CharmParams{
			Name: "wordpress",
		}),
	})
	uri2 := secrets.NewURI()
	cp := state.CreateSecretParams{
		Version: 1,
		Owner:   owner2.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err := s.store.CreateSecret(uri2, cp)
	c.Assert(err, jc.ErrorIsNil)

	uri3 := secrets.NewURI()
	cp = state.CreateSecretParams{
		Version: 1,
		Owner:   s.ownerUnit.Tag(),
		UpdateSecretParams: state.UpdateSecretParams{
			LeaderToken: &fakeToken{},
			Data:        map[string]string{"foo": "bar"},
		},
	}
	_, err = s.store.CreateSecret(uri3, cp)
	c.Assert(err, jc.ErrorIsNil)

	_, err = s.store.DeleteSecret(uri)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(uri.String())
	wc.AssertNoChange()

	_, err = s.store.DeleteSecret(uri2)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertNoChange()

	_, err = s.store.DeleteSecret(uri3)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(uri3.String())
	wc.AssertNoChange()
}

func (s *SecretsObsoleteWatcherSuite) TestWatchDeletedSupercedesObsolete(c *gc.C) {
	w, uri := s.setupWatcher(c)
	wc := testing.NewStringsWatcherC(c, w)
	defer testing.AssertStop(c, w)

	err := s.State.SaveSecretConsumer(uri, names.NewApplicationTag("foo"), &secrets.SecretConsumerMetadata{
		CurrentRevision: 1,
	})
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertNoChange()

	p := state.UpdateSecretParams{
		LeaderToken: &fakeToken{},
		Data:        map[string]string{"foo": "bar2"},
	}
	_, err = s.store.UpdateSecret(uri, p)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertNoChange()

	err = s.State.SaveSecretConsumer(uri, names.NewApplicationTag("foo2"), &secrets.SecretConsumerMetadata{
		CurrentRevision: 2,
	})
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertNoChange()

	// The previous consumer of rev 1 now uses rev 2; rev 1 is orphaned.
	err = s.State.SaveSecretConsumer(uri, names.NewApplicationTag("foo"), &secrets.SecretConsumerMetadata{
		CurrentRevision: 2,
	})
	c.Assert(err, jc.ErrorIsNil)

	// Deleting the secret removes any pending orphaned changes.
	_, err = s.store.DeleteSecret(uri)
	c.Assert(err, jc.ErrorIsNil)
	wc.AssertChange(uri.String())
	wc.AssertNoChange()
}
