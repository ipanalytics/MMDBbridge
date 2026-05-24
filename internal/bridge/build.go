package bridge

import (
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/ipanalytics/mmdbbridge/internal/schema"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

func Build(s *schema.Schema, csvPath, outPath string) error {
	in, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open csv %s: %w", csvPath, err)
	}
	defer in.Close()

	reader := csv.NewReader(in)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read csv header: %w", err)
	}
	cols := columnIndex(header)
	if _, ok := cols[s.Network.Column]; !ok {
		return fmt.Errorf("csv missing network column %q", s.Network.Column)
	}
	for path, field := range s.Fields {
		if _, ok := cols[field.Column]; !ok {
			return fmt.Errorf("csv missing column %q for field %s", field.Column, path)
		}
	}

	writer, err := mmdbwriter.New(mmdbwriter.Options{
		BuildEpoch:              s.Metadata.BuildEpoch,
		DatabaseType:            s.Metadata.DatabaseType,
		Description:             map[string]string{"en": s.Metadata.Description},
		IPVersion:               s.Metadata.IPVersion,
		RecordSize:              s.Metadata.RecordSize,
		Languages:               []string{"en"},
		IncludeReservedNetworks: s.Metadata.IncludeReservedNetworks,
		DisableIPv4Aliasing:     true,
	})
	if err != nil {
		return fmt.Errorf("create mmdb writer: %w", err)
	}

	rowNum := 1
	inserted := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		rowNum++
		if err != nil {
			return fmt.Errorf("read csv row %d: %w", rowNum, err)
		}
		networkValue := cell(row, cols[s.Network.Column])
		ipNet, err := parseNetwork(networkValue)
		if err != nil {
			return fmt.Errorf("row %d network %q: %w", rowNum, networkValue, err)
		}
		record, err := buildRecord(s, row, cols)
		if err != nil {
			return fmt.Errorf("row %d: %w", rowNum, err)
		}
		if err := writer.Insert(ipNet, record); err != nil {
			return fmt.Errorf("insert row %d %s: %w", rowNum, ipNet, err)
		}
		inserted++
	}
	if inserted == 0 {
		return fmt.Errorf("csv has no data rows")
	}

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create mmdb %s: %w", outPath, err)
	}
	defer out.Close()
	if _, err := writer.WriteTo(out); err != nil {
		return fmt.Errorf("write mmdb %s: %w", outPath, err)
	}
	return nil
}

func columnIndex(header []string) map[string]int {
	cols := make(map[string]int, len(header))
	for i, h := range header {
		cols[strings.TrimSpace(h)] = i
	}
	return cols
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func parseNetwork(v string) (*net.IPNet, error) {
	if strings.Contains(v, "/") {
		ip, network, err := net.ParseCIDR(v)
		if err != nil {
			return nil, err
		}
		network.IP = ip.Mask(network.Mask)
		return network, nil
	}
	ip := net.ParseIP(v)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP or CIDR")
	}
	if ip.To4() != nil {
		return &net.IPNet{IP: ip.To4(), Mask: net.CIDRMask(32, 32)}, nil
	}
	return &net.IPNet{IP: ip.To16(), Mask: net.CIDRMask(128, 128)}, nil
}

func buildRecord(s *schema.Schema, row []string, cols map[string]int) (mmdbtype.Map, error) {
	record := mmdbtype.Map{}
	for _, path := range s.OrderedFieldPaths() {
		field := s.Fields[path]
		raw := cell(row, cols[field.Column])
		if raw == "" && field.Required {
			return nil, fmt.Errorf("required column %q is empty", field.Column)
		}
		if raw == "" {
			continue
		}
		value, err := typedValue(raw, field)
		if err != nil {
			return nil, fmt.Errorf("column %q: %w", field.Column, err)
		}
		setPath(record, path, value)
	}
	return record, nil
}

func typedValue(raw string, field schema.FieldConfig) (mmdbtype.DataType, error) {
	switch field.Type {
	case "string":
		return mmdbtype.String(raw), nil
	case "bool":
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, err
		}
		return mmdbtype.Bool(v), nil
	case "int32":
		v, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return nil, err
		}
		return mmdbtype.Int32(int32(v)), nil
	case "uint16":
		v, err := strconv.ParseUint(raw, 10, 16)
		if err != nil {
			return nil, err
		}
		return mmdbtype.Uint16(uint16(v)), nil
	case "uint32":
		v, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return nil, err
		}
		return mmdbtype.Uint32(uint32(v)), nil
	case "uint64":
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return nil, err
		}
		return mmdbtype.Uint64(v), nil
	case "float32":
		v, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return nil, err
		}
		return mmdbtype.Float32(float32(v)), nil
	case "float64":
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, err
		}
		return mmdbtype.Float64(v), nil
	case "string_array":
		split := field.Split
		if split == "" {
			split = ","
		}
		parts := strings.Split(raw, split)
		out := make(mmdbtype.Slice, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, mmdbtype.String(part))
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", field.Type)
	}
}

func setPath(record mmdbtype.Map, path string, value mmdbtype.DataType) {
	parts := strings.Split(path, ".")
	current := record
	for i, part := range parts {
		key := mmdbtype.String(part)
		if i == len(parts)-1 {
			current[key] = value
			return
		}
		next, ok := current[key].(mmdbtype.Map)
		if !ok {
			next = mmdbtype.Map{}
			current[key] = next
		}
		current = next
	}
}
