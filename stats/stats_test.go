package stats

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("no Caller information")
	}
	testDataDir := filepath.Join(filepath.Dir(filename), "..", "testdata")
	if err := os.Chdir(testDataDir); err != nil {
		panic("failed to change working dir to " + testDataDir)
	}
	orig, err := os.Open(filepath.Join(testDataDir, "initial.sqlite"))
	if err != nil {
		panic("failed to open initial.sqlite")
	}
	dbData, err := io.ReadAll(orig)
	if err != nil {
		panic("failed to read initial.sqlite")
	}
	dest := filepath.Join(testDataDir, "testing.sqlite")
	err = os.WriteFile(dest, dbData, 0644)
	if err != nil {
		panic("failed to write testing.sqlite")
	}
	os.Exit(m.Run())
}

func TestGetAllNoFilters(t *testing.T) {
	db, err := NewDatabase("./testing.sqlite")
	assert.NoError(t, err)
	recs, err := db.GetAllRecords(ListFilter{})
	assert.NoError(t, err)
	assert.Equal(t, 8, len(recs))
}

func TestGetAllOnlyTestExcluded(t *testing.T) {
	db, err := NewDatabase("./testing.sqlite")
	assert.NoError(t, err)
	recs, err := db.GetAllRecords(ListFilter{}.SetTrainingExcluded(true))
	assert.NoError(t, err)
	assert.Equal(t, 4, len(recs))
	for _, v := range recs {
		assert.Equal(t, true, v.TrainingExclude)
	}
}

func TestGetAllOnlyTestIncluded(t *testing.T) {
	db, err := NewDatabase("./testing.sqlite")
	assert.NoError(t, err)
	recs, err := db.GetAllRecords(ListFilter{}.SetTrainingExcluded(false))
	assert.NoError(t, err)
	assert.Equal(t, 4, len(recs))
	for _, v := range recs {
		assert.Equal(t, false, v.TrainingExclude)
	}
}

func TestGetAllOnlyBenchmarked(t *testing.T) {
	db, err := NewDatabase("./testing.sqlite")
	assert.NoError(t, err)
	recs, err := db.GetAllRecords(ListFilter{}.SetBenchmarked(true))
	assert.NoError(t, err)
	assert.Equal(t, 4, len(recs))
	for _, v := range recs {
		assert.NotEqual(t, 0.0, v.BenchTime)
	}
}

func TestGetAllOnlyNonBenchmarked(t *testing.T) {
	db, err := NewDatabase("./testing.sqlite")
	assert.NoError(t, err)
	recs, err := db.GetAllRecords(ListFilter{}.SetBenchmarked(false))
	assert.NoError(t, err)
	assert.Equal(t, 4, len(recs))
	for _, v := range recs {
		assert.Equal(t, 0.0, v.BenchTime)
	}
}
