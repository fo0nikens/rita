package structure

import (
	"github.com/ocmdev/rita/config"
	"github.com/ocmdev/rita/database"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// BuildHostsCollection builds the 'host' collection for this timeframe.
// Runs via mongodb aggregation. Sourced from the 'conn' table.
func BuildHostsCollection(res *database.Resources) {
	// Create the aggregate command
	sourceCollectionName,
		newCollectionName,
		newCollectionKeys,
		pipeline := getHosts(res.System)

	// Aggregate it!
	errorCheck := res.DB.CreateCollection(newCollectionName, false, newCollectionKeys)
	if errorCheck != nil {
		res.Log.Error("Failed: ", newCollectionName, errorCheck)
		return
	}

	ssn := res.DB.Session.Copy()
	defer ssn.Close()

	res.DB.AggregateCollection(sourceCollectionName, ssn, pipeline)
}

func getHosts(sysCfg *config.SystemConfig) (string, string, []mgo.Index, []bson.D) {
	// Name of source collection which will be aggregated into the new collection
	sourceCollectionName := sysCfg.StructureConfig.ConnTable

	// Name of the new collection
	newCollectionName := sysCfg.StructureConfig.HostTable

	// Desired indeces
	keys := []mgo.Index{
		{Key: []string{"ip"}, Unique: true},
		{Key: []string{"local"}},
	}

	// Aggregation script
	// nolint: vet
	pipeline := []bson.D{
		{
			{"$project", bson.D{
				{"hosts", []interface{}{
					bson.D{
						{"ip", "$id_origin_h"},
						{"local", "$local_orig"},
					},
					bson.D{
						{"ip", "$id_resp_h"},
						{"local", "$local_resp"},
					},
				}},
			}},
		},
		{
			{"$unwind", "$hosts"},
		},
		{
			{"$group", bson.D{
				{"_id", "$hosts.ip"},
				{"local", bson.D{
					{"$first", "$hosts.local"},
				}},
			}},
		},
		{
			{"$project", bson.D{
				{"_id", 0},
				{"ip", "$_id"},
				{"local", 1},
			}},
		},
		{
			{"$out", newCollectionName},
		},
	}

	return sourceCollectionName, newCollectionName, keys, pipeline
}
