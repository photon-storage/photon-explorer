package server

import (
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/photon-storage/photon-explorer/api/pagination"
)

func TestValidateFunc(t *testing.T) {
	testCases := []struct {
		name    string
		fn      handleFunc
		wantErr bool
	}{
		{
			name:    "the interface is not function type",
			fn:      10,
			wantErr: true,
		},
		{
			name:    "the first input parameter of the func isn't gin.Context type",
			fn:      func(i int) {},
			wantErr: true,
		},
		{
			name:    "the second input parameter of the func isn't a pointer type",
			fn:      func(c *gin.Context, i int) {},
			wantErr: true,
		},
		{
			name:    "the third input parameter of the func isn't a pagination type",
			fn:      func(c *gin.Context, i *int, page pagination.Query) {},
			wantErr: true,
		},
		{
			name:    "missing return values",
			fn:      func(c *gin.Context, i *int, page *pagination.Query) {},
			wantErr: true,
		},
		{
			name: "the last return value of the func must be an error type",
			fn: func(c *gin.Context, i *int, page *pagination.Query) int {
				return 0
			},
			wantErr: true,
		},
		{
			name: "the first return value of the func must " +
				"be a paginationResp type",
			fn: func(c *gin.Context, i *int, page *pagination.Query) (int, error) {
				return 0, nil
			},
			wantErr: true,
		},
		{
			name: "one input parameter of the function",
			fn: func(c *gin.Context) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "two input parameters of the function",
			fn: func(c *gin.Context, i *int) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "three input parameters of the function",
			fn: func(
				c *gin.Context,
				i *int,
				page *pagination.Query,
			) (*pagination.Result, error) {
				return nil, nil
			},
			wantErr: false,
		},
	}
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			if err := validateFunc(c.fn); (err != nil) != c.wantErr {
				t.Errorf("validate func return error = %v,"+
					" want error %v", err, c.wantErr)
			}
		})
	}
}
