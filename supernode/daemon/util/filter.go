package util

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	errorType "github.com/dragonflyoss/Dragonfly/common/errors"
	"github.com/dragonflyoss/Dragonfly/common/util"

	"github.com/pkg/errors"
)

const (
	// PAGENUM identity the page number of the data.
	// The default value is 0.
	PAGENUM = "pageNum"

	// PAGESIZE identity the page size of the data.
	// If this value equals 0, return all values.
	PAGESIZE = "pageSize"

	// SORTKEY identity the sort key of the data.
	// Each mgr needs to define acceptable values based on its own implementation.
	SORTKEY = "sortKey"

	// SORTDIRECT identity the sort direct of the data.
	// The value can only be ASC or DESC.
	SORTDIRECT = "sortDirect"

	// ASCDIRECT means to sort the records in ascending order.
	ASCDIRECT = "ASC"

	// DESCDIRECT means to sort the records in descending order.
	DESCDIRECT = "DESC"
)

var sortDirectMap = map[string]bool{
	ASCDIRECT:  true,
	DESCDIRECT: true,
}

// PageFilter is a struct.
type PageFilter struct {
	PageNum    int
	PageSize   int
	SortKey    []string
	SortDirect string
}

// ParseFilter gets filter params from request and returns a map[string][]string.
func ParseFilter(req *http.Request, sortKeyMap map[string]bool) (pageFilter *PageFilter, err error) {
	v := req.URL.Query()
	pageFilter = &PageFilter{}

	// pageNum
	pageNum, err := stoi(v.Get(PAGENUM))
	if err != nil {
		return nil, errors.Wrapf(errorType.ErrInvalidValue, "pageNum %s is not a number: %v", pageNum, err)
	}
	pageFilter.PageNum = pageNum

	// pageSize
	pageSize, err := stoi(v.Get(PAGESIZE))
	if err != nil {
		return nil, errors.Wrapf(errorType.ErrInvalidValue, "pageSize %s is not a number: %v", pageSize, err)
	}
	pageFilter.PageSize = pageSize

	// sortDirect
	sortDirect := v.Get(SORTDIRECT)
	if sortDirect == "" {
		sortDirect = ASCDIRECT
	}
	pageFilter.SortDirect = sortDirect

	// sortKey
	if sortKey, ok := v[SORTKEY]; ok {
		pageFilter.SortKey = sortKey
	}

	err = ValidateFilter(pageFilter, sortKeyMap)
	if err != nil {
		return nil, err
	}

	return
}

func stoi(str string) (int, error) {
	if util.IsEmptyStr(str) {
		return 0, nil
	}

	result, err := strconv.Atoi(str)
	if err != nil || result < 0 {
		return -1, err
	}
	return result, nil
}

// ValidateFilter validates the param of filter.
// The caller should  customize the sortKeyMap which specifies the sort keys it supports.
func ValidateFilter(pageFilter *PageFilter, sortKeyMap map[string]bool) error {
	// pageNum
	if pageFilter.PageNum < 0 {
		return errors.Wrapf(errorType.ErrInvalidValue, "pageNum %s is not a natural number: %v", pageFilter.PageNum)
	}

	// pageSize
	if pageFilter.PageSize < 0 {
		return errors.Wrapf(errorType.ErrInvalidValue, "pageSize %s is not a natural number: %v", pageFilter.PageSize)
	}

	// sortDirect
	if _, ok := sortDirectMap[strings.ToUpper(pageFilter.SortDirect)]; !ok {
		return errors.Wrapf(errorType.ErrInvalidValue, "unexpected sortDirect %s", pageFilter.SortDirect)
	}

	// sortKey
	if len(pageFilter.SortKey) == 0 || sortKeyMap == nil {
		return nil
	}
	for _, value := range pageFilter.SortKey {
		if v, ok := sortKeyMap[value]; !ok || !v {
			return errors.Wrapf(errorType.ErrInvalidValue, "unexpected sortKey %s", value)
		}
	}

	return nil
}

// GetPageValues gets some pages of metaSlice after ordering it.
// The less is a function that reports whether the element with
// index i should sort before the element with index j.
//
// Eg:
// people := []struct {
//     Name string
//     Age  int
// }{
//     {"Gopher", 7},
//     {"Alice", 55},
//     {"Vera", 24},
//     {"Bob", 75},
// }
//
// If you want to sort it by age, and the less function should be defined as follows:
//
// less := func(i, j int) bool { return people[i].Age < people[j].Age }
func GetPageValues(metaSlice []interface{}, pageNum, pageSize int,
	less func(i, j int) bool) []interface{} {

	if metaSlice == nil {
		return nil
	}
	if less == nil {
		return metaSlice
	}

	// sort the data slice
	sort.Slice(metaSlice, less)

	if pageSize == 0 {
		return metaSlice
	}

	sliceLength := len(metaSlice)
	start := pageNum * pageSize
	end := (pageNum + 1) * pageSize

	if sliceLength < start {
		return nil
	}
	if sliceLength < end {
		return metaSlice[start:sliceLength]
	}
	return metaSlice[start:end]
}

// IsDESC returns whether the sortDirect is desc.
func IsDESC(str string) bool {
	return strings.ToUpper(str) == DESCDIRECT
}
