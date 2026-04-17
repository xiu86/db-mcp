//go:build !nomongo
// +build !nomongo

package driver

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"db-mcp/internal/config"
	"db-mcp/internal/errors"
	"db-mcp/internal/sanitizer"
	"db-mcp/pkg/logger"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDriver implements the MongoDB driver
type MongoDriver struct {
	client   *mongo.Client
	database *mongo.Database
	config   *config.MongoConfig
	logger   *logger.Logger
}

// NewMongoDriver creates a new MongoDB driver
func NewMongoDriver(cfg *config.MongoConfig, log *logger.Logger) (*MongoDriver, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := cfg.URI
	if uri == "" {
		// Build URI with optional authentication
		if cfg.Username != "" && cfg.Password != "" {
			authSource := cfg.AuthSource
			if authSource == "" {
				authSource = cfg.Database
			}
			uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=%s",
				url.QueryEscape(cfg.Username),
				url.QueryEscape(cfg.Password),
				cfg.Host,
				cfg.Port,
				cfg.Database,
				authSource,
			)
		} else {
			uri = fmt.Sprintf("mongodb://%s:%d", cfg.Host, cfg.Port)
		}
	}

	clientOptions := options.Client().ApplyURI(uri)
	if cfg.MaxPoolSize > 0 {
		clientOptions.SetMaxPoolSize(cfg.MaxPoolSize)
	}
	if cfg.MinPoolSize > 0 {
		clientOptions.SetMinPoolSize(cfg.MinPoolSize)
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &MongoDriver{
		client:   client,
		database: client.Database(cfg.Database),
		config:   cfg,
		logger:   log,
	}, nil
}

// Ping checks the connection
func (d *MongoDriver) Ping(ctx context.Context) error {
	return d.client.Ping(ctx, nil)
}

// Close closes the connection
func (d *MongoDriver) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return d.client.Disconnect(ctx)
}

// DriverType returns the driver type
func (d *MongoDriver) DriverType() DriverType {
	return DriverMongoDB
}

// CurrentDatabase returns the current database
func (d *MongoDriver) CurrentDatabase() string {
	return d.config.Database
}

// UseDatabase switches to the specified database
func (d *MongoDriver) UseDatabase(database string) error {
	if database == "" {
		return errors.NewError(errors.ErrInvalidInput, "database name cannot be empty", nil)
	}
	if err := sanitizer.ValidateTableName(database); err != nil {
		return errors.NewError(errors.ErrInvalidInput, "invalid database name", err)
	}
	d.database = d.client.Database(database)
	d.config.Database = database
	return nil
}

// convertWhere converts unified where conditions to MongoDB filter
func convertWhere(where map[string]interface{}) bson.M {
	filter := bson.M{}

	for key, value := range where {
		// Handle operators
		if m, ok := value.(map[string]interface{}); ok {
			for op, v := range m {
				switch op {
				case "$in":
					filter[key] = bson.M{"$in": v}
				case "$nin":
					filter[key] = bson.M{"$nin": v}
				case "$gt":
					filter[key] = bson.M{"$gt": v}
				case "$gte":
					filter[key] = bson.M{"$gte": v}
				case "$lt":
					filter[key] = bson.M{"$lt": v}
				case "$lte":
					filter[key] = bson.M{"$lte": v}
				case "$ne":
					filter[key] = bson.M{"$ne": v}
				case "$like":
					filter[key] = bson.M{"$regex": v}
				case "$between":
					// $between: [min, max]
					if arr, ok := v.([]interface{}); ok && len(arr) >= 2 {
						filter[key] = bson.M{"$gte": arr[0], "$lte": arr[1]}
					}
				default:
					// Unknown operator, use directly
					filter[key] = v
				}
			}
		} else {
			filter[key] = value
		}
	}

	return filter
}

// convertOrder converts OrderBy to MongoDB sort
func convertOrder(order []OrderBy) bson.D {
	if len(order) == 0 {
		return nil
	}

	var sort bson.D
	for _, o := range order {
		dir := 1
		if o.Direction == "desc" || o.Direction == "DESC" {
			dir = -1
		}
		sort = append(sort, bson.E{Key: o.Field, Value: dir})
	}
	return sort
}

// convertFields converts fields to MongoDB projection
func convertFields(fields []string) bson.M {
	if len(fields) == 0 {
		return nil
	}

	projection := bson.M{}
	for _, f := range fields {
		projection[f] = 1
	}
	return projection
}

// Query executes a find query
func (d *MongoDriver) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, errors.NewError(errors.ErrInvalidInput, "invalid collection name", err)
	}
	collection := d.database.Collection(req.Table)

	filter := convertWhere(req.Where)
	opts := options.Find()

	if fields := convertFields(req.Fields); fields != nil {
		opts.SetProjection(fields)
	}

	if sort := convertOrder(req.Order); sort != nil {
		opts.SetSort(sort)
	}

	if req.Limit > 0 {
		opts.SetLimit(int64(req.Limit))
	}
	if req.Offset > 0 {
		opts.SetSkip(int64(req.Offset))
	}

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, errors.NewError(errors.ErrDatabase, "query failed", err)
	}
	defer cursor.Close(ctx)

	var rows []map[string]interface{}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, errors.NewError(errors.ErrDatabase, "decode results failed", err)
	}

	// Count total
	total, _ := collection.CountDocuments(ctx, filter)

	return &QueryResult{
		Rows:    rows,
		Total:   total,
		Message: "Query successful",
	}, nil
}

// Insert inserts a document
func (d *MongoDriver) Insert(ctx context.Context, req *InsertRequest) (*MutationResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, errors.NewError(errors.ErrInvalidInput, "invalid collection name", err)
	}
	collection := d.database.Collection(req.Table)

	_, err := collection.InsertOne(ctx, req.Data)
	if err != nil {
		return nil, errors.NewError(errors.ErrDatabase, "insert failed", err)
	}

	return &MutationResult{
		AffectedRows: 1,
		LastInsertID:  1,
		Message:       "Insert successful",
	}, nil
}

// Update updates documents
func (d *MongoDriver) Update(ctx context.Context, req *UpdateRequest) (*MutationResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, errors.NewError(errors.ErrInvalidInput, "invalid collection name", err)
	}
	collection := d.database.Collection(req.Table)

	filter := convertWhere(req.Where)
	update := bson.M{"$set": req.Data}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return nil, errors.NewError(errors.ErrDatabase, "update failed", err)
	}

	return &MutationResult{
		AffectedRows: result.ModifiedCount,
		Message:      "Update successful",
	}, nil
}

// Delete deletes data (logical or physical delete)
func (d *MongoDriver) Delete(ctx context.Context, req *DeleteRequest) (*MutationResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, errors.NewError(errors.ErrInvalidInput, "invalid collection name", err)
	}
	collection := d.database.Collection(req.Table)
	filter := convertWhere(req.Where)

	// If DeleteField is set, perform logical delete
	if req.DeleteField != nil && len(req.DeleteField.Fields) > 0 {
		updates := bson.M{}
		for _, field := range req.DeleteField.Fields {
			updates[field.Name] = field.TrueValue
		}
		result, err := collection.UpdateMany(ctx, filter, bson.M{"$set": updates})
		if err != nil {
			return nil, errors.NewError(errors.ErrDatabase, "delete failed", err)
		}
		return &MutationResult{
			AffectedRows: result.ModifiedCount,
			Message:      "Delete successful",
		}, nil
	} else {
		// Physical delete
		result, err := collection.DeleteMany(ctx, filter)
		if err != nil {
			return nil, errors.NewError(errors.ErrDatabase, "delete failed", err)
		}
		return &MutationResult{
			AffectedRows: result.DeletedCount,
			Message:      "Delete successful",
		}, nil
	}
}

// BatchInsert inserts multiple documents
func (d *MongoDriver) BatchInsert(ctx context.Context, req *BatchInsertRequest) (*BatchResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, errors.NewError(errors.ErrInvalidInput, "invalid collection name", err)
	}
	if len(req.Data) == 0 {
		return &BatchResult{
			SuccessCount: 0,
			FailedCount:  0,
		}, nil
	}

	collection := d.database.Collection(req.Table)

	// Convert to []interface{}
	docs := make([]interface{}, len(req.Data))
	for i, d := range req.Data {
		docs[i] = d
	}

	result, err := collection.InsertMany(ctx, docs)
	if err != nil {
		// If batch insert fails, try inserting one by one
		return d.batchInsertOneByOne(ctx, collection, req.Data)
	}

	return &BatchResult{
		SuccessCount: int64(len(result.InsertedIDs)),
		FailedCount:  0,
	}, nil
}

// batchInsertOneByOne inserts documents one by one (when batch insert fails)
func (d *MongoDriver) batchInsertOneByOne(ctx context.Context, collection *mongo.Collection, data []map[string]interface{}) (*BatchResult, error) {
	var successCount, failedCount int64
	var batchErrors []BatchError

	for i, doc := range data {
		_, err := collection.InsertOne(ctx, doc)
		if err != nil {
			failedCount++
			batchErrors = append(batchErrors, BatchError{Index: i, Message: err.Error()})
		} else {
			successCount++
		}
	}

	return &BatchResult{
		SuccessCount: successCount,
		FailedCount:  failedCount,
		Errors:       batchErrors,
	}, nil
}

// BatchUpdate updates multiple documents
func (d *MongoDriver) BatchUpdate(ctx context.Context, req *BatchUpdateRequest) (*BatchResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, errors.NewError(errors.ErrInvalidInput, "invalid collection name", err)
	}
	collection := d.database.Collection(req.Table)

	keyField := req.KeyField
	if keyField == "" {
		keyField = "_id"
	}

	var successCount, failedCount int64
	var batchErrors []BatchError

	for i, data := range req.Data {
		keyValue, exists := data[keyField]
		if !exists {
			failedCount++
			batchErrors = append(batchErrors, BatchError{Index: i, Message: "key field value is nil"})
			continue
		}

		filter := bson.M{keyField: keyValue}
		update := bson.M{"$set": data}

		result, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			failedCount++
			batchErrors = append(batchErrors, BatchError{Index: i, Message: err.Error()})
		} else {
			successCount++
			_ = result // suppress unused warning
		}
	}

	return &BatchResult{
		SuccessCount: successCount,
		FailedCount:  failedCount,
		Errors:       batchErrors,
	}, nil
}

// BatchDelete deletes multiple documents
func (d *MongoDriver) BatchDelete(ctx context.Context, req *BatchDeleteRequest) (*BatchResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, errors.NewError(errors.ErrInvalidInput, "invalid collection name", err)
	}
	collection := d.database.Collection(req.Table)

	idField := req.IDField
	if idField == "" {
		idField = "_id"
	}

	var successCount, failedCount int64
	var batchErrors []BatchError

	for i, id := range req.IDs {
		filter := bson.M{idField: id}

		var err error
		if req.DeleteField != nil && len(req.DeleteField.Fields) > 0 {
			// Logical delete
			updates := bson.M{}
			for _, field := range req.DeleteField.Fields {
				updates[field.Name] = field.TrueValue
			}
			_, err = collection.UpdateOne(ctx, filter, bson.M{"$set": updates})
		} else {
			// Physical delete
			_, err = collection.DeleteOne(ctx, filter)
		}

		if err != nil {
			failedCount++
			batchErrors = append(batchErrors, BatchError{Index: i, Message: err.Error()})
		} else {
			successCount++
		}
	}

	return &BatchResult{
		SuccessCount: successCount,
		FailedCount:  failedCount,
		Errors:       batchErrors,
	}, nil
}

// JoinQuery MongoDB does not support native Join, uses Aggregation $lookup
func (d *MongoDriver) JoinQuery(ctx context.Context, req *JoinRequest) (*QueryResult, error) {
	if len(req.Tables) < 2 {
		return nil, errors.NewError(errors.ErrInvalidInput, "at least 2 collections required for join", nil)
	}
	if len(req.Tables) > 5 {
		return nil, errors.NewError(errors.ErrInvalidInput, "join exceeds maximum of 5 collections", nil)
	}

	// Validate all collection names
	for i, tbl := range req.Tables {
		if err := sanitizer.ValidateTableName(tbl.Name); err != nil {
			return nil, errors.NewError(errors.ErrInvalidInput,
				fmt.Sprintf("invalid collection name at index %d: %s", i, tbl.Name), err)
		}
	}

	// MongoDB uses Aggregation Pipeline to simulate Join
	pipeline := mongo.Pipeline{}

	// $lookup for each join
	for _, join := range req.Joins {
		lookupStage := bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: join.ToTable},
			{Key: "localField", Value: join.FromField},
			{Key: "foreignField", Value: join.ToField},
			{Key: "as", Value: join.ToTable + "_lookup"},
		}}}
		pipeline = append(pipeline, lookupStage)

		// $unwind to flatten the lookup result
		unwindStage := bson.D{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$" + join.ToTable + "_lookup"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}}
		pipeline = append(pipeline, unwindStage)
	}

	// Where/filter
	if len(req.Where) > 0 {
		matchStage := bson.D{{Key: "$match", Value: convertWhere(req.Where)}}
		pipeline = append(pipeline, matchStage)
	}

	// Projection
	if fields := convertFields(req.Fields); fields != nil {
		projectStage := bson.D{{Key: "$project", Value: fields}}
		pipeline = append(pipeline, projectStage)
	}

	// Sort
	if sort := convertOrder(req.Order); sort != nil {
		sortStage := bson.D{{Key: "$sort", Value: sort}}
		pipeline = append(pipeline, sortStage)
	}

	// Limit
	if req.Limit > 0 {
		limitStage := bson.D{{Key: "$limit", Value: req.Limit}}
		pipeline = append(pipeline, limitStage)
	}

	collection := d.database.Collection(req.Tables[0].Name)
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.NewError(errors.ErrDatabase, "join query failed", err)
	}
	defer cursor.Close(ctx)

	var rows []map[string]interface{}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, errors.NewError(errors.ErrDatabase, "decode join results failed", err)
	}

	return &QueryResult{
		Rows:    rows,
		Total:   int64(len(rows)),
		Message: "Join query successful",
	}, nil
}

// GetTableSchema gets MongoDB collection schema (returns collection info)
func (d *MongoDriver) GetTableSchema(collectionName string) (*TableSchema, error) {
	if err := sanitizer.ValidateTableName(collectionName); err != nil {
		return nil, errors.NewError(errors.ErrInvalidInput, "invalid collection name", err)
	}
	ctx := context.Background()

	// Get collection index info as schema
	indexes, err := d.database.Collection(collectionName).Indexes().List(ctx)
	if err != nil {
		return nil, errors.NewError(errors.ErrDatabase, "failed to get collection indexes", err)
	}
	defer indexes.Close(ctx)

	var columns []ColumnInfo

	// MongoDB is schema-less, we return index info as table "schema"
	for indexes.Next(ctx) {
		var index bson.M
		if err := indexes.Decode(&index); err == nil {
			key, ok := index["key"].(bson.M)
			if ok {
				for name, _ := range key {
					columns = append(columns, ColumnInfo{
						Name:       name,
						DataType:   "indexed",
						IsNullable: "yes",
						ColumnKey:  "MUL",
						Comment:    "Indexed field",
					})
				}
			}
		}
	}

	// Add _id field
	columns = append([]ColumnInfo{{
		Name:       "_id",
		DataType:   "ObjectId",
		IsNullable: "no",
		ColumnKey:  "PRI",
		Comment:    "Primary key",
	}}, columns...)

	return &TableSchema{
		TableName: collectionName,
		Columns:   columns,
	}, nil
}
