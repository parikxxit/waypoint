package pagination

import (
	"errors"
	"fmt"

	pb "github.com/hashicorp/waypoint/pkg/server/gen"
	paginatorV2 "github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	gormV2 "gorm.io/gorm"
)

// GormV2CursorPaginator wraps a GORM cursor paginator and adds methods to
// implement both the Paginator and GormV2Paginator interfaces.
//
// An example of a generated pagination query when using SortFields CreateAt and
// UUID with a pagination limit of 2 is:
//
//	SELECT * FROM "organizations"  WHERE {user-specified}
//	AND (organizations.created_at < '2020-03-24 14:16:42' OR organizations.created_at = '2020-03-24 14:16:42'
//	     AND organizations.uuid < '11ea6dda-15c7-6480-8d5d-acde48001122')
//	ORDER BY organizations.created_at DESC, organizations.uuid DESC LIMIT 3;
type GormV2CursorPaginator struct {
	*paginatorV2.Paginator
	cursor paginatorV2.Cursor
}

// NewGormV2CursorPaginator takes a request context and returns a
// GormV2CursorPaginator.
func NewGormV2CursorPaginator(ctx *RequestContext) (*GormV2CursorPaginator, error) {
	if ctx == nil {
		return nil, errors.New("nil request context passed")
	}

	keyRules := []paginatorV2.Rule{}
	for _, f := range ctx.SortFields {
		rule := paginatorV2.Rule{}
		rule.Key = f.Field

		switch f.Order {
		case Ascending, unsetOrder:
			rule.Order = paginatorV2.ASC
		case Descending:
			rule.Order = paginatorV2.DESC
		}

		keyRules = append(keyRules, rule)
	}

	opts := []paginatorV2.Option{
		&paginatorV2.Config{
			Rules:  keyRules,
			Limit:  int(ctx.Limit),
			After:  ctx.Cursor.Next.GetGormPagination(),
			Before: ctx.Cursor.Previous.GetGormPagination(),
		},
	}

	return &GormV2CursorPaginator{
		Paginator: paginatorV2.New(opts...),
	}, nil
}

// Type returns the paginator type.
func (gc *GormV2CursorPaginator) Type() PaginatorType {
	return PaginatorGormCursor
}

// Cursor returns the internal pagination cursor. Only valid after Paginate has
// been called.
func (gc *GormV2CursorPaginator) Cursor() *pb.PaginationCursor {
	c := gc.cursor
	pc := &pb.PaginationCursor{}
	if c.After != nil {
		pc.Next = &pb.PaginationCursor_Cursor{
			Value: &pb.PaginationCursor_Cursor_GormPagination{
				GormPagination: *c.After,
			},
		}
	}

	if c.Before != nil {
		pc.Previous = &pb.PaginationCursor_Cursor{
			Value: &pb.PaginationCursor_Cursor_GormPagination{
				GormPagination: *c.Before,
			},
		}
	}

	return pc

}

// PaginationResponse returns the public pagination response. It is only valid
// after Paginate has been called.
func (gc *GormV2CursorPaginator) PaginationResponse() *pb.PaginationResponse {
	resp, err := createResponse(gc.Cursor())
	if err != nil {
		panic(fmt.Sprintf("impossible error: %v", err))
	}

	return resp
}

// Paginate encapsulate the new paginator.Paginate(db, dest) interface
func (gc *GormV2CursorPaginator) Paginate(db *gormV2.DB, dest interface{}) (result *gormV2.DB) {
	result, cursor, err := gc.Paginator.Paginate(db, dest)
	gc.cursor = cursor

	// For validation and decoding errors in Paginate, 'result' will be nil.
	// In this case, we return the given 'db' and append the error in 'err' to db.Error
	// so that the error can be check from the returned 'result'
	if result == nil {
		result = db
	}

	// Append err, if any, to result.Error
	_ = result.AddError(err)
	return result
}