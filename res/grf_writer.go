package res

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

type GRFPackStats struct {
	Files int
	Bytes int64
}

type grfPackEntry struct {
	name        string
	packedSize  uint32
	alignedSize uint32
	realSize    uint32
	offset      uint32
}

func PackGRF(outputPath, root string) (GRFPackStats, error) {
	files, err := collectGRFFiles(root)
	if err != nil {
		return GRFPackStats{}, err
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return GRFPackStats{}, err
	}
	defer out.Close()

	header := make([]byte, grfHeaderSize)
	copy(header[:15], []byte("Master of Magic"))
	binary.LittleEndian.PutUint32(header[34:38], 0)
	binary.LittleEndian.PutUint32(header[38:42], uint32(len(files)+7))
	binary.LittleEndian.PutUint32(header[42:46], grfVersion200)
	if _, err := out.Write(header); err != nil {
		return GRFPackStats{}, err
	}

	var entries []grfPackEntry
	var stats GRFPackStats
	for _, grfName := range files {
		path := filepath.Join(root, filepath.FromSlash(grfName))
		data, err := os.ReadFile(path)
		if err != nil {
			return GRFPackStats{}, err
		}
		compressed, err := zlibCompress(data)
		if err != nil {
			return GRFPackStats{}, err
		}
		offset, err := out.Seek(0, io.SeekCurrent)
		if err != nil {
			return GRFPackStats{}, err
		}
		if offset < grfHeaderSize || offset-grfHeaderSize > int64(^uint32(0)) {
			return GRFPackStats{}, fmt.Errorf("%s: GRF offset overflow", grfName)
		}
		if _, err := out.Write(compressed); err != nil {
			return GRFPackStats{}, err
		}
		entries = append(entries, grfPackEntry{
			name:        grfName,
			packedSize:  uint32(len(compressed)),
			alignedSize: uint32(len(compressed)),
			realSize:    uint32(len(data)),
			offset:      uint32(offset - grfHeaderSize),
		})
		stats.Files++
		stats.Bytes += int64(len(data))
	}

	table, err := buildGRFTable(entries)
	if err != nil {
		return GRFPackStats{}, err
	}
	compressedTable, err := zlibCompress(table)
	if err != nil {
		return GRFPackStats{}, err
	}
	tableOffset, err := out.Seek(0, io.SeekCurrent)
	if err != nil {
		return GRFPackStats{}, err
	}
	if tableOffset-grfHeaderSize > int64(^uint32(0)) {
		return GRFPackStats{}, fmt.Errorf("GRF table offset overflow")
	}
	if err := binary.Write(out, binary.LittleEndian, uint32(len(compressedTable))); err != nil {
		return GRFPackStats{}, err
	}
	if err := binary.Write(out, binary.LittleEndian, uint32(len(table))); err != nil {
		return GRFPackStats{}, err
	}
	if _, err := out.Write(compressedTable); err != nil {
		return GRFPackStats{}, err
	}
	if _, err := out.Seek(30, io.SeekStart); err != nil {
		return GRFPackStats{}, err
	}
	if err := binary.Write(out, binary.LittleEndian, uint32(tableOffset-grfHeaderSize)); err != nil {
		return GRFPackStats{}, err
	}
	return stats, out.Close()
}

func collectGRFFiles(root string) ([]string, error) {
	var files []string
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	return files, err
}

func buildGRFTable(entries []grfPackEntry) ([]byte, error) {
	var table bytes.Buffer
	for _, entry := range entries {
		name, err := encodeGRFTableName(entry.name)
		if err != nil {
			return nil, err
		}
		table.WriteString(name)
		table.WriteByte(0)
		writeU32Buffer(&table, entry.packedSize)
		writeU32Buffer(&table, entry.alignedSize)
		writeU32Buffer(&table, entry.realSize)
		table.WriteByte(0x01)
		writeU32Buffer(&table, entry.offset)
	}
	return table.Bytes(), nil
}

func encodeGRFTableName(name string) (string, error) {
	name = strings.ReplaceAll(filepath.ToSlash(name), "/", "\\")
	encoded, _, err := transform.String(korean.EUCKR.NewEncoder(), name)
	if err == nil {
		return encoded, nil
	}
	// Extracted trees often carry CP1252 mojibake names (EUC-KR bytes the
	// extraction tool decoded as Windows-1252). Windows-1252 maps those
	// runes back to the original bytes, so the archive keeps the exact
	// names the game looks up.
	encoded, _, err = transform.String(charmap.Windows1252.NewEncoder(), name)
	if err != nil {
		return "", err
	}
	return encoded, nil
}

func zlibCompress(data []byte) ([]byte, error) {
	var out bytes.Buffer
	writer := zlib.NewWriter(&out)
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func writeU32Buffer(buf *bytes.Buffer, value uint32) {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], value)
	buf.Write(tmp[:])
}
