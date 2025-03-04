package rpc

import (
	"github.com/qubic/go-archiver/protobuff"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_V2Endpoints_createPaginationInfo(t *testing.T) {

	pagination, err := getPaginationInformation(0, 1, 100)
	assert.NoError(t, err)
	verify(t, pagination,
		0, 0, 100, 0, -1, -1)

	pagination, err = getPaginationInformation(1, 1, 100)
	assert.NoError(t, err)
	verify(t, pagination,
		1, 1, 100, 1, -1, -1)

	pagination, err = getPaginationInformation(12345, 1, 100)
	assert.NoError(t, err)
	verify(t, pagination,
		12345, 1, 100, 124, -1, 2)

	pagination, err = getPaginationInformation(12345, 10, 100)
	assert.NoError(t, err)
	verify(t, pagination,
		12345, 10, 100, 124, 9, 11)

	pagination, err = getPaginationInformation(12345, 124, 100)
	assert.NoError(t, err)
	verify(t, pagination,
		12345, 124, 100, 124, 123, -1)

	pagination, err = getPaginationInformation(12345, 125, 100)
	assert.NoError(t, err)
	verify(t, pagination,
		12345, 125, 100, 124, 124, -1)

	pagination, err = getPaginationInformation(12345, 999, 100)
	assert.NoError(t, err)
	verify(t, pagination,
		12345, 999, 100, 124, 124, -1)

}

func Test_V2Endpoints_createPaginationInfo_givenInvalidArguments_thenError(t *testing.T) {
	_, err := getPaginationInformation(12345, 0, 100)
	assert.Error(t, err)

	_, err = getPaginationInformation(-1, 1, 100)
	assert.Error(t, err)

	_, err = getPaginationInformation(12345, 1, 0)
	assert.Error(t, err)
}

func verify(t *testing.T, pagination *protobuff.Pagination, totalRecords, pageNumber, pageSize, totalPages, previousPage, nextPage int) {
	assert.NotNil(t, pagination)
	assert.Equal(t, int32(totalRecords), pagination.TotalRecords, "unexpected number of total records")
	assert.Equal(t, int32(pageNumber), pagination.CurrentPage, "unexpected current page number")
	assert.Equal(t, int32(pageSize), pagination.PageSize, "unexpected page size")
	assert.Equal(t, int32(totalPages), pagination.TotalPages, "unexpected number of total pages")
	assert.Equal(t, int32(previousPage), pagination.PreviousPage, "unexpected previous page number")
	assert.Equal(t, int32(nextPage), pagination.NextPage, "unexpected next page number")
}
