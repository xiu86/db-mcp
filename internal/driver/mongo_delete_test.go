//go:build !nomongo

package driver

import (
	"context"
	"testing"

	"db-mcp/internal/detector"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestMongoDeleteModeCommands(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).CollectionName("users"))
	for _, tc := range []struct {
		name        string
		physical    bool
		softField   bool
		wantCommand string
	}{
		{"logical uses update", false, true, "update"},
		{"missing logical field never deletes", false, false, ""},
		{"explicit physical uses delete", true, false, "delete"},
		{"physical ignores logical field", true, true, "delete"},
	} {
		for _, batch := range []bool{false, true} {
			name := tc.name
			if batch {
				name += " batch"
			}
			mt.Run(name, func(mt *mtest.T) {
				d := &MongoDriver{database: mt.DB}
				mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
				var field *detector.DeleteFieldInfo
				if tc.softField {
					field = &detector.DeleteFieldInfo{Fields: []detector.Field{{Name: "is_deleted", TrueValue: "1"}}}
				}

				var err error
				if batch {
					req := &BatchDeleteRequest{PhysicalDelete: tc.physical, Table: "users", IDs: []string{"user-1"}, IDField: "id", DeleteField: field}
					_, err = d.BatchDelete(context.Background(), req)
				} else {
					req := &DeleteRequest{PhysicalDelete: tc.physical, Table: "users", Where: map[string]interface{}{"id": "user-1"}, DeleteField: field}
					_, err = d.Delete(context.Background(), req)
				}
				if tc.wantCommand == "" {
					require.ErrorContains(mt, err, "no delete field detected")
					require.Nil(mt, mt.GetStartedEvent())
					return
				}
				require.NoError(mt, err)
				event := mt.GetStartedEvent()
				require.NotNil(mt, event)
				require.Equal(mt, tc.wantCommand, event.CommandName)
				array := "updates"
				if tc.physical {
					array = "deletes"
				}
				commands, err := event.Command.Lookup(array).Array().Values()
				require.NoError(mt, err)
				require.Len(mt, commands, 1)
				require.Equal(mt, "user-1", commands[0].Document().Lookup("q").Document().Lookup("id").StringValue())
			})
		}
	}
}

func TestMongoPhysicalDeleteRejectsEmptyFilter(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	for _, tc := range []struct {
		name  string
		where map[string]interface{}
	}{
		{"empty where", map[string]interface{}{}},
		{"empty nested predicate", map[string]interface{}{"id": map[string]interface{}{}}},
		{"invalid range predicate", map[string]interface{}{"id": map[string]interface{}{"$between": []interface{}{}}}},
	} {
		mt.Run(tc.name, func(mt *mtest.T) {
			d := &MongoDriver{database: mt.DB}
			mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 3}))
			_, err := d.Delete(context.Background(), &DeleteRequest{Table: "users", Where: tc.where, PhysicalDelete: true})
			require.ErrorContains(mt, err, "physical deletion requires non-empty")
			require.Nil(mt, mt.GetStartedEvent())
		})
	}
}
