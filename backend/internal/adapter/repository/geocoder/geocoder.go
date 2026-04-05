package geocoder

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed regions.json
var regionsData []byte

type region struct {
	Code string  `json:"code"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type Repository struct {
	byCode map[string]region
	byName map[string]region
}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) Init() error {
	var regions []region
	if err := json.Unmarshal(regionsData, &regions); err != nil {
		return err
	}

	r.byCode = make(map[string]region, len(regions))
	r.byName = make(map[string]region, len(regions))

	for _, reg := range regions {
		r.byCode[reg.Code] = reg
		r.byName[strings.ToLower(reg.Name)] = reg
	}

	return nil
}

func (r *Repository) Resolve(cadastral string, oblast string) (float64, float64) {
	if reg, ok := r.parseCadastral(cadastral); ok {
		return reg.Lat, reg.Lon
	}

	if reg, ok := r.resolveByName(oblast); ok {
		return reg.Lat, reg.Lon
	}

	return 51.13, 69.41
}

func (r *Repository) parseCadastral(cadastral string) (region, bool) {
	clean := strings.TrimSpace(strings.ToUpper(cadastral))

	if len(clean) >= 4 && strings.HasPrefix(clean, "KZ") {
		code := clean[2:4]
		if reg, ok := r.byCode[code]; ok {
			return reg, true
		}
	}

	parts := strings.SplitN(clean, "-", 2)
	if len(parts) >= 1 && len(parts[0]) >= 4 && strings.HasPrefix(parts[0], "KZ") {
		code := parts[0][2:4]
		if reg, ok := r.byCode[code]; ok {
			return reg, true
		}
	}

	return region{}, false
}

func (r *Repository) resolveByName(oblast string) (region, bool) {
	key := strings.ToLower(strings.TrimSpace(oblast))
	if key == "" {
		return region{}, false
	}

	if reg, ok := r.byName[key]; ok {
		return reg, true
	}

	for k, reg := range r.byName {
		if strings.Contains(key, k) || strings.Contains(k, key) {
			return reg, true
		}
	}

	return region{}, false
}
