package maps

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/res"
)

func (m *System) RuntimeRSMModel(manager *res.Manager, modelPath string) *res.RSM {
	if manager == nil || modelPath == "" {
		return nil
	}
	if m.runtimeRSMModels == nil {
		m.runtimeRSMModels = make(map[string]*res.RSM)
	}
	if rsm, ok := m.runtimeRSMModels[modelPath]; ok {
		return rsm
	}
	rsm, err := LoadRSMModel(manager, modelPath)
	if err != nil {
		glog.Warnf("runtime rsm model unavailable model=%s: %v", modelPath, err)
		m.runtimeRSMModels[modelPath] = nil
		return nil
	}
	m.runtimeRSMModels[modelPath] = rsm
	return rsm
}

func LoadGAT(manager *res.Manager, mapName string) (*res.GAT, string, error) {
	if manager == nil {
		return nil, "", fmt.Errorf("no game resources available for map %s", mapName)
	}
	base := strings.TrimSuffix(strings.TrimSuffix(mapName, ".gat"), ".rsw")
	candidates := []string{
		"data\\" + base + ".gat",
		"data/" + base + ".gat",
		base + ".gat",
	}
	data, source, err := manager.ReadFileCandidates(candidates)
	if err != nil {
		return nil, source, fmt.Errorf("cannot load GAT for map %s: %w", mapName, err)
	}
	gat, err := res.ParseGAT(data)
	if err != nil {
		return nil, source, fmt.Errorf("parse %s: %w", source, err)
	}
	return gat, source, nil
}

func LoadRSW(manager *res.Manager, mapName string) (*res.RSW, string, error) {
	base := strings.TrimSuffix(strings.TrimSuffix(mapName, ".gat"), ".rsw")
	candidates := []string{
		"data\\" + base + ".rsw",
		"data/" + base + ".rsw",
		base + ".rsw",
	}
	data, source, err := manager.ReadFileCandidates(candidates)
	if err != nil {
		return nil, source, fmt.Errorf("cannot load RSW for map %s: %w", mapName, err)
	}
	rsw, err := res.ParseRSW(data)
	return rsw, source, err
}

func LoadRSMModels(manager *res.Manager, rsw *res.RSW) (map[string]*res.RSM, int) {
	if rsw == nil {
		return nil, 0
	}
	models := make(map[string]*res.RSM)
	failures := 0
	for _, placement := range rsw.Models {
		if placement.Filename == "" {
			continue
		}
		if _, ok := models[placement.Filename]; ok {
			continue
		}
		rsm, err := LoadRSMModel(manager, placement.Filename)
		models[placement.Filename] = rsm
		if err != nil {
			failures++
			continue
		}
	}
	return models, failures
}

func LoadRSMModel(manager *res.Manager, filename string) (*res.RSM, error) {
	var data []byte
	for _, candidate := range res.RSMModelCandidates(filename) {
		raw, err := manager.ReadFile(candidate)
		if err == nil {
			data = raw
			break
		}
	}
	if data == nil {
		return nil, fmt.Errorf("rsm not found: %s", filename)
	}
	return res.ParseRSM(data)
}
