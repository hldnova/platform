package functions_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/influxdata/platform"
	"github.com/influxdata/platform/mock"
	"github.com/influxdata/platform/query"
	_ "github.com/influxdata/platform/query/builtin"
	"github.com/influxdata/platform/query/csv"
	"github.com/influxdata/platform/query/influxql"
	"github.com/influxdata/platform/query/querytest"

	"github.com/andreyvit/diff"
)

var dbrpMappingSvc = mock.NewDBRPMappingService()

func init() {
	mapping := platform.DBRPMapping{
		Cluster:         "cluster",
		Database:        "db0",
		RetentionPolicy: "autogen",
		Default:         true,
		OrganizationID:  platform.ID("org"),
		BucketID:        platform.ID("bucket"),
	}
	dbrpMappingSvc.FindByFn = func(ctx context.Context, cluster string, db string, rp string) (*platform.DBRPMapping, error) {
		return &mapping, nil
	}
	dbrpMappingSvc.FindFn = func(ctx context.Context, filter platform.DBRPMappingFilter) (*platform.DBRPMapping, error) {
		return &mapping, nil
	}
	dbrpMappingSvc.FindManyFn = func(ctx context.Context, filter platform.DBRPMappingFilter, opt ...platform.FindOptions) ([]*platform.DBRPMapping, int, error) {
		return []*platform.DBRPMapping{&mapping}, 1, nil
	}
}

var skipTests = map[string]string{
	"derivative":                "derivative not supported by influxql (https://github.com/influxdata/platform/issues/93)",
	"filter_by_tags":            "arbitrary filtering not supported by influxql (https://github.com/influxdata/platform/issues/94)",
	"window_group_mean_ungroup": "error in influxql: failed to run query: timeValue column \"_start\" does not exist (https://github.com/influxdata/platform/issues/97)",
	"string_max":                "error: invalid use of function: *functions.MaxSelector has no implementation for type string (https://github.com/influxdata/platform/issues/224)",
	"null_as_value":             "null not supported as value in influxql (https://github.com/influxdata/platform/issues/353)",
	"difference_panic":          "difference() panics when no table is supplied",
	"string_interp":             "string interpolation not working as expected in flux (https://github.com/influxdata/platform/issues/404)",
}

func Test_QueryEndToEnd(t *testing.T) {
	qs := querytest.GetQueryServiceBridge()

	influxqlTranspiler := influxql.NewTranspiler(dbrpMappingSvc)

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "testdata")

	fluxFiles, err := filepath.Glob(filepath.Join(path, "*.flux"))
	if err != nil {
		t.Fatalf("error searching for Flux files: %s", err)
	}

	for _, fluxFile := range fluxFiles {
		ext := filepath.Ext(fluxFile)
		prefix := fluxFile[0 : len(fluxFile)-len(ext)]

		_, caseName := filepath.Split(prefix)
		if reason, ok := skipTests[caseName]; ok {
			t.Run(caseName, func(t *testing.T) {
				t.Skip(reason)
			})
			continue
		}

		fluxName := caseName + ".flux"
		influxqlName := caseName + ".influxql"
		t.Run(fluxName, func(t *testing.T) {
			queryTester(t, qs, prefix, ".flux")
		})
		t.Run(influxqlName, func(t *testing.T) {
			queryTranspileTester(t, influxqlTranspiler, qs, prefix, ".influxql")
		})
	}
}

func queryTester(t *testing.T, qs query.QueryService, prefix, queryExt string) {
	q, err := querytest.GetTestData(prefix, queryExt)
	if err != nil {
		t.Fatal(err)
	}

	csvOut, err := querytest.GetTestData(prefix, ".out.csv")
	if err != nil {
		t.Fatal(err)
	}

	spec, err := query.Compile(context.Background(), q, time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	csvIn := prefix + ".in.csv"
	enc := csv.NewMultiResultEncoder(csv.DefaultEncoderConfig())

	QueryTestCheckSpec(t, qs, spec, csvIn, csvOut, enc)
}

func queryTranspileTester(t *testing.T, transpiler query.Transpiler, qs query.QueryService, prefix, queryExt string) {
	q, err := querytest.GetTestData(prefix, queryExt)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("query missing")
		} else {
			t.Fatal(err)
		}
	}

	csvOut, err := querytest.GetTestData(prefix, ".out.csv")
	if err != nil {
		t.Fatal(err)
	}

	spec, err := transpiler.Transpile(context.Background(), q)
	if err != nil {
		t.Fatalf("failed to transpile: %v", err)
	}

	csvIn := prefix + ".in.csv"
	enc := csv.NewMultiResultEncoder(csv.DefaultEncoderConfig())
	QueryTestCheckSpec(t, qs, spec, csvIn, csvOut, enc)

	enc = influxql.NewMultiResultEncoder()
	jsonOut, err := querytest.GetTestData(prefix, ".out.json")
	if err != nil {
		t.Logf("skipping json evaluation: %s", err)
		return
	}
	QueryTestCheckSpec(t, qs, spec, csvIn, jsonOut, enc)
}

func QueryTestCheckSpec(t *testing.T, qs query.QueryService, spec *query.Spec, inputFile, want string, enc query.MultiResultEncoder) {
	t.Helper()

	querytest.ReplaceFromSpec(spec, inputFile)

	got, err := querytest.GetQueryEncodedResults(qs, spec, inputFile, enc)
	if err != nil {
		t.Errorf("failed to run query: %v", err)
		return
	}

	if g, w := strings.TrimSpace(got), strings.TrimSpace(want); g != w {
		t.Errorf("result not as expected want(-) got (+):\n%v", diff.LineDiff(w, g))
	}
}
