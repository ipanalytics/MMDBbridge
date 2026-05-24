package bridge

import (
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ipanalytics/mmdbbridge/internal/schema"
	"github.com/oschwald/maxminddb-golang"
)

func Export(s *schema.Schema, mmdbPath, outPath string) error {
	db, err := maxminddb.Open(mmdbPath)
	if err != nil {
		return fmt.Errorf("open mmdb %s: %w", mmdbPath, err)
	}
	defer db.Close()

	rows := [][]string{s.Header()}
	networks := db.Networks(maxminddb.SkipAliasedNetworks)
	for networks.Next() {
		var record map[string]any
		network, err := networks.Network(&record)
		if err != nil {
			return fmt.Errorf("read network: %w", err)
		}
		rows = append(rows, exportRow(s, network, record))
	}
	if err := networks.Err(); err != nil {
		return fmt.Errorf("iterate mmdb: %w", err)
	}
	sort.Slice(rows[1:], func(i, j int) bool {
		return rows[i+1][0] < rows[j+1][0]
	})

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create csv %s: %w", outPath, err)
	}
	defer out.Close()
	writer := csv.NewWriter(out)
	if err := writer.WriteAll(rows); err != nil {
		return fmt.Errorf("write csv %s: %w", outPath, err)
	}
	return writer.Error()
}

func exportRow(s *schema.Schema, network *net.IPNet, record map[string]any) []string {
	row := []string{network.String()}
	for _, path := range s.OrderedFieldPaths() {
		field := s.Fields[path]
		row = append(row, formatValue(getPath(record, path), field))
	}
	return row
}

func getPath(record map[string]any, path string) any {
	var current any = record
	for _, part := range strings.Split(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}

func formatValue(v any, field schema.FieldConfig) string {
	if v == nil {
		return ""
	}
	switch field.Type {
	case "string":
		return fmt.Sprint(v)
	case "bool":
		switch b := v.(type) {
		case bool:
			return strconv.FormatBool(b)
		default:
			return fmt.Sprint(v)
		}
	case "int32", "uint16", "uint32", "uint64":
		return formatInteger(v)
	case "float32", "float64":
		return strconv.FormatFloat(toFloat64(v), 'f', -1, 64)
	case "string_array":
		split := field.Split
		if split == "" {
			split = ","
		}
		switch values := v.(type) {
		case []any:
			parts := make([]string, 0, len(values))
			for _, item := range values {
				parts = append(parts, fmt.Sprint(item))
			}
			return strings.Join(parts, split)
		case []string:
			return strings.Join(values, split)
		default:
			return fmt.Sprint(v)
		}
	default:
		return fmt.Sprint(v)
	}
}

func formatInteger(v any) string {
	switch n := v.(type) {
	case uint:
		return strconv.FormatUint(uint64(n), 10)
	case uint16:
		return strconv.FormatUint(uint64(n), 10)
	case uint32:
		return strconv.FormatUint(uint64(n), 10)
	case uint64:
		return strconv.FormatUint(n, 10)
	case int:
		return strconv.FormatInt(int64(n), 10)
	case int32:
		return strconv.FormatInt(int64(n), 10)
	case int64:
		return strconv.FormatInt(n, 10)
	case float32:
		return strconv.FormatFloat(float64(n), 'f', 0, 32)
	case float64:
		return strconv.FormatFloat(n, 'f', 0, 64)
	default:
		return fmt.Sprint(v)
	}
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case uint:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	default:
		f, _ := strconv.ParseFloat(fmt.Sprint(v), 64)
		return f
	}
}
