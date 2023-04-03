package pagination

import (
	"fmt"
	"github.com/hashicorp/waypoint/pkg/pagination/database/testsql"
	"testing"

	gormV2 "github.com/hashicorp/waypoint/pkg/pagination/database/gorm"
	pb "github.com/hashicorp/waypoint/pkg/server/gen"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGormV2CursorPaginator_Invalid(t *testing.T) {
	r := require.New(t)
	_, err := NewGormV2CursorPaginator(nil)
	r.Error(err)
}

func TestGormV2CursorPaginator_Valid(t *testing.T) {
	dbV1 := testsql.TestPostgresDBWithOpts(t, "paginator_v2", &testsql.TestDBOptions{
		SkipMigration: true,
	})
	dbV1.AutoMigrate(&order{})
	db, err := gormV2.ConvertToV2(dbV1)
	require.NoError(t, err)

	r := require.New(t)
	config := mockConfig()
	config.DefaultSortedFields = []SortedField{
		{Field: "CreatedAt", Order: Descending},
		{Field: "ID"},
	}
	p, err := NewGormV2CursorPaginator(&RequestContext{

		Cursor:     &pb.PaginationCursor{},
		Paginator:  PaginatorGormCursor,
		Limit:      10,
		SortFields: config.DefaultSortedFields,
	})
	r.NoError(err)
	r.NotNil(p)
	r.EqualValues(PaginatorGormCursor, p.Type())

	// Insert some orders
	newOrdersV2(t, db, 22)

	// Paginate
	var out []*order
	result := p.Paginate(db, &out)
	r.NoError(result.Error)
	r.Len(out, 10)
	r.True(out[0].CreatedAt.After(out[1].CreatedAt))

	// Get the cursor
	c := p.Cursor()
	r.NotNil(c)
	r.NotNil(c.Next)
	r.Nil(c.Previous)

	// Get the response
	resp := p.PaginationResponse()
	r.NotNil(resp)
	r.NotEmpty(resp.NextPageToken)
	r.Empty(resp.PreviousPageToken)

	// Page again
	p, err = NewGormV2CursorPaginator(&RequestContext{

		Cursor:     p.Cursor(),
		Paginator:  PaginatorGormCursor,
		Limit:      10,
		SortFields: config.DefaultSortedFields,
	})
	r.NoError(err)

	var out2 []*order
	result = p.Paginate(db, &out2)
	r.NoError(result.Error)
	r.Len(out2, 10)
	r.True(out2[0].CreatedAt.After(out2[1].CreatedAt))

	// Get the cursor
	c2 := p.Cursor()
	r.NotNil(c2)
	r.NotNil(c2.Next)
	r.NotNil(c2.Previous)

	// Get the response
	resp2 := p.PaginationResponse()
	r.NotNil(resp2)
	r.NotEmpty(resp2.NextPageToken)
	r.NotEmpty(resp2.PreviousPageToken)

	// Page again and clear the previous token
	c2.Previous = nil
	p, err = NewGormV2CursorPaginator(&RequestContext{
		Cursor:     c2,
		Paginator:  PaginatorGormCursor,
		Limit:      10,
		SortFields: config.DefaultSortedFields,
	})
	r.NoError(err)

	var out3 []*order
	result = p.Paginate(db, &out3)
	r.NoError(result.Error)
	r.Len(out3, 2)
	r.True(out3[0].CreatedAt.After(out3[1].CreatedAt))

	// Get the cursor
	c3 := p.Cursor()
	r.NotNil(c3)
	r.Nil(c3.Next)
	r.NotNil(c3.Previous)

	// Get the response
	resp3 := p.PaginationResponse()
	r.NotNil(resp3)
	r.Empty(resp3.NextPageToken)
	r.NotEmpty(resp3.PreviousPageToken)

	// Page back
	p, err = NewGormV2CursorPaginator(&RequestContext{
		Cursor:     c3,
		Paginator:  PaginatorGormCursor,
		Limit:      10,
		SortFields: config.DefaultSortedFields,
	})
	r.NoError(err)

	var out4 []*order
	result = p.Paginate(db, &out4)
	r.NoError(result.Error)
	r.Len(out4, 10)
	r.True(out4[0].CreatedAt.After(out4[1].CreatedAt))
	r.Equal(out2, out4) // should be equal do page 2

	// Get the cursor
	c4 := p.Cursor()
	r.NotNil(c4)
	r.NotNil(c4.Next)
	r.NotNil(c4.Previous)

	// Get the response
	resp4 := p.PaginationResponse()
	r.NotNil(resp4)
	r.NotEmpty(resp4.NextPageToken)
	r.NotEmpty(resp4.PreviousPageToken)
}

func TestGormV2CursorPaginator_PageBackWithQuery(t *testing.T) {
	dbV1 := testsql.TestPostgresDBWithOpts(t, "paginator_v2", &testsql.TestDBOptions{
		SkipMigration: true,
	})
	dbV1.AutoMigrate(&order{})
	db, err := gormV2.ConvertToV2(dbV1)
	require.NoError(t, err)

	// Insert some orders
	newOrdersV2(t, db, 22)

	r := require.New(t)
	sortedFields := []SortedField{
		{Field: "ID"},
		{Field: "CreatedAt", Order: Descending},
	}

	query := db.Where("price = ?", 123) // should have 11 elements

	p, err := NewGormV2CursorPaginator(&RequestContext{
		Cursor:     &pb.PaginationCursor{},
		Limit:      5,
		SortFields: sortedFields,
	})
	r.NoError(err)
	r.NotNil(p)
	r.EqualValues(PaginatorGormCursor, p.Type())

	// Page 1
	var out []order
	result := p.Paginate(query, &out)
	r.NoError(result.Error)
	r.Len(out, 5)
	r.True(out[0].ID < out[1].ID)

	// Get the cursor
	c := p.Cursor()
	r.NotNil(c)
	r.NotNil(c.Next)
	r.Nil(c.Previous)

	// Get the response
	resp := p.PaginationResponse()
	r.NotNil(resp)
	r.NotEmpty(resp.NextPageToken)
	r.Empty(resp.PreviousPageToken)

	query = db.Where("price = ?", 123) // should have 11 elements

	// Paginate to Page 2
	p, err = NewGormV2CursorPaginator(&RequestContext{
		Cursor:     c,
		Limit:      5,
		SortFields: sortedFields,
	})
	r.NoError(err)

	var out2 []order
	result = p.Paginate(query, &out2)
	r.NoError(result.Error)
	r.Len(out2, 5)
	r.True(out2[0].ID < out2[1].ID)
	r.Equal(out[4].ID+1, out2[0].ID) // IDs should be sequential

	// Get the cursor
	c2 := p.Cursor()
	r.NotNil(c2)
	r.NotNil(c2.Next)
	r.NotNil(c2.Previous)

	// Get the response
	resp2 := p.PaginationResponse()
	r.NotNil(resp2)
	r.NotEmpty(resp2.NextPageToken)
	r.NotEmpty(resp2.PreviousPageToken)

	query = db.Where("price = ?", 123) // should have 11 elements

	// Page back to page 1
	c2.Next = nil
	p, err = NewGormV2CursorPaginator(&RequestContext{
		Cursor:     c2,
		Paginator:  PaginatorGormCursor,
		Limit:      5,
		SortFields: sortedFields,
	})
	r.NoError(err)

	var out3 []order
	result = p.Paginate(query, &out3)
	r.NoError(result.Error)
	r.Len(out3, 5)
	r.True(out3[0].ID < out3[1].ID)
	r.Equal(out, out3) // Should be equal to page 1

	// Get the cursor
	c3 := p.Cursor()
	r.NotNil(c3)
	r.NotNil(c3.Next)
	r.Nil(c3.Previous)

	// Get the response
	resp3 := p.PaginationResponse()
	r.NotNil(resp3)
	r.NotEmpty(resp3.NextPageToken)
	r.Empty(resp3.PreviousPageToken)
}

func newOrdersV2(t *testing.T, db *gorm.DB, n int) []order {
	orders := make([]order, n)
	for i := 0; i < n; i++ {
		price := 456
		if i <= n/2 {
			price = 123
		}
		orders[i] = order{ID: i + 1, Name: fmt.Sprintf("order_%d", i), Price: price}
		if err := db.Create(&orders[i]).Error; err != nil {
			t.Fatal(err.Error())
		}
	}
	return orders
}