// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"sort"

	"github.com/juju/mgo/v3"
	"github.com/juju/mgo/v3/bson"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/utils/v3"
	"github.com/kr/pretty"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/core/crossmodel"
	"github.com/juju/juju/storage/provider"
	coretesting "github.com/juju/juju/testing"
)

type upgradesSuite struct {
	internalStateSuite
}

var _ = gc.Suite(&upgradesSuite{})

//nolint:unused
type expectUpgradedData struct {
	coll     *mgo.Collection
	expected []bson.M
	filter   bson.D
}

//nolint:unused
func upgradedData(coll *mgo.Collection, expected []bson.M) expectUpgradedData {
	return expectUpgradedData{
		coll:     coll,
		expected: expected,
	}
}

//nolint:unused
func (s *upgradesSuite) assertUpgradedData(c *gc.C, upgrade func(*StatePool) error, expect ...expectUpgradedData) {
	// Two rounds to check idempotency.
	for i := 0; i < 2; i++ {
		c.Logf("Run: %d", i)
		err := upgrade(s.pool)
		c.Assert(err, jc.ErrorIsNil)

		for _, expect := range expect {
			var docs []bson.M
			err = expect.coll.Find(expect.filter).Sort("_id").All(&docs)
			c.Assert(err, jc.ErrorIsNil)
			for i, d := range docs {
				doc := d
				delete(doc, "txn-queue")
				delete(doc, "txn-revno")
				delete(doc, "version")
				docs[i] = doc
			}
			c.Assert(docs, jc.DeepEquals, expect.expected,
				gc.Commentf("differences: %s", pretty.Diff(docs, expect.expected)))
		}
	}
}

//nolint:unused
func (s *upgradesSuite) makeModel(c *gc.C, name string, attr coretesting.Attrs) *State {
	uuid := utils.MustNewUUID()
	cfg := coretesting.CustomModelConfig(c, coretesting.Attrs{
		"name": name,
		"uuid": uuid.String(),
	}.Merge(attr))
	m, err := s.state.Model()
	c.Assert(err, jc.ErrorIsNil)
	_, st, err := s.controller.NewModel(ModelArgs{
		Type:                    ModelTypeIAAS,
		CloudName:               "dummy",
		CloudRegion:             "dummy-region",
		Config:                  cfg,
		Owner:                   m.Owner(),
		StorageProviderRegistry: provider.CommonStorageProviders(),
	})
	c.Assert(err, jc.ErrorIsNil)
	return st
}

type bsonMById []bson.M

func (x bsonMById) Len() int { return len(x) }

func (x bsonMById) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x bsonMById) Less(i, j int) bool {
	return x[i]["_id"].(string) < x[j]["_id"].(string)
}

func (s *upgradesSuite) TestConvertApplicationOfferTokenKeys(c *gc.C) {
	st := s.state
	remoteEntitiesColl, closer := st.db().GetRawCollection(remoteEntitiesC)
	defer closer()

	_, err := st.AddUser("bob", "", "shhhh", "admin")
	c.Assert(err, jc.ErrorIsNil)
	app := AddTestingApplication(c, st, "mysql", AddTestingCharm(c, st, "mysql"))
	offer1, err := NewApplicationOffers(st).AddOffer(crossmodel.AddApplicationOfferArgs{
		OfferName:       "myoffer1",
		Owner:           "bob",
		ApplicationName: app.Name(),
	})
	c.Assert(err, jc.ErrorIsNil)
	offer2, err := NewApplicationOffers(st).AddOffer(crossmodel.AddApplicationOfferArgs{
		OfferName:       "myoffer2",
		Owner:           "bob",
		ApplicationName: app.Name(),
	})
	c.Assert(err, jc.ErrorIsNil)

	err = remoteEntitiesColl.Insert(bson.M{
		"_id":        ensureModelUUID(st.ModelUUID(), "applicationoffer-myoffer1"),
		"model-uuid": st.ModelUUID(),
		"token":      "token1",
	})
	c.Assert(err, jc.ErrorIsNil)
	err = remoteEntitiesColl.Insert(bson.M{
		"_id":        ensureModelUUID(st.ModelUUID(), "applicationoffer-myoffer2"),
		"model-uuid": st.ModelUUID(),
		"token":      "token2",
	})
	c.Assert(err, jc.ErrorIsNil)
	err = remoteEntitiesColl.Insert(bson.M{
		"_id":        ensureModelUUID(st.ModelUUID(), "application-myapp1"),
		"model-uuid": st.ModelUUID(),
		"token":      "apptoken1",
	})
	c.Assert(err, jc.ErrorIsNil)

	var expected bsonMById
	expected = append(expected, bson.M{
		"_id":        ensureModelUUID(st.ModelUUID(), "applicationoffer-"+offer1.OfferUUID),
		"model-uuid": st.ModelUUID(),
		"token":      "token1",
	}, bson.M{
		"_id":        ensureModelUUID(st.ModelUUID(), "applicationoffer-"+offer2.OfferUUID),
		"model-uuid": st.ModelUUID(),
		"token":      "token2",
	}, bson.M{
		"_id":        ensureModelUUID(st.ModelUUID(), "application-myapp1"),
		"model-uuid": st.ModelUUID(),
		"token":      "apptoken1",
	})

	sort.Sort(expected)
	expectedData := upgradedData(remoteEntitiesColl, expected)
	s.assertUpgradedData(c, ConvertApplicationOfferTokenKeys, expectedData)
}
