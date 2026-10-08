package main

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Toughness: health / damage in percent for each size group (100 = unchanged).
type Toughness struct {
	Health  map[string]int `json:"health,omitempty"`
	Damage  map[string]int `json:"damage,omitempty"`
	HealthM map[string]int `json:"health_m,omitempty"` // per machine, multiplied with its group
	DamageM map[string]int `json:"damage_m,omitempty"`
}

var toughGroups = []string{"Small", "Medium", "Large", "Flyer", "Other"}

type toughData struct {
	DMType      string      `json:"dm_type"`
	DM          string      `json:"dm"`
	DDType      string      `json:"dd_type"`
	DD          string      `json:"dd"`
	HealthTypes [][2]string `json:"health_types"`
	DLC         []string    `json:"dlc"`
}

func (t Toughness) active() bool {
	for _, m := range []map[string]int{t.Health, t.Damage, t.HealthM, t.DamageM} {
		for _, v := range m {
			if v != 0 && v != 100 {
				return true
			}
		}
	}
	return false
}

func pct(m map[string]int, g string) int {
	v := m[g]
	if v <= 0 {
		return 100
	}
	if v < 10 {
		v = 10
	}
	if v > 500 {
		v = 500
	}
	return v
}

func (c MapConfig) groupOf(m string) string {
	switch c.tier()[m] {
	case 1:
		return "Small"
	case 2:
		return "Medium"
	case 3:
		return "Large"
	case 9:
		return "Flyer"
	}
	return "Other"
}

func coreStr(s string) []byte {
	b := make([]byte, 8, 8+len(s))
	binary.LittleEndian.PutUint32(b[0:], uint32(len(s)))
	binary.LittleEndian.PutUint32(b[4:], decimaCRC(s))
	return append(b, s...)
}

func f32(v float32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, math.Float32bits(v))
	return b
}

// damage taken modifier: multiplier on every damage type the player deals (health = 1/multiplier)
func buildDamageTaken(td toughData, id []byte, name string, mult float32) ([]byte, error) {
	tpl, err := hex.DecodeString(td.DM)
	if err != nil {
		return nil, err
	}
	r := &reader{tpl, 16}
	r.str()
	after := r.o // f32 mult, f32 cap, u8, u32, u32 count, refs..., tail
	mid := tpl[after+4 : after+4+4+1+4]
	r.o = after + 4 + 4 + 1 + 4
	n := int(r.u32())
	for i := 0; i < n; i++ {
		r.ref()
	}
	tail := tpl[r.o:]
	out := append([]byte{}, id...)
	out = append(out, coreStr(name)...)
	out = append(out, f32(mult)...)
	out = append(out, mid...)
	var cnt [4]byte
	binary.LittleEndian.PutUint32(cnt[:], uint32(len(td.HealthTypes)))
	out = append(out, cnt[:]...)
	for _, t := range td.HealthTypes {
		u, err := parseUUIDLE(t[1])
		if err != nil {
			return nil, err
		}
		out = append(out, 2)
		out = append(out, u...)
		out = append(out, coreStr("entities/damagetypes")...)
	}
	return append(out, tail...), nil
}

// damage dealt modifier (same layout the game uses on bandit heavies)
func buildDamageDealt(td toughData, id []byte, name string, mult float32) ([]byte, error) {
	tpl, err := hex.DecodeString(td.DD)
	if err != nil {
		return nil, err
	}
	r := &reader{tpl, 16}
	r.str()
	rest := append([]byte{}, tpl[r.o:]...) // u32 u32 u32(count=0) u32 f32 mult ...
	if len(rest) < 20 || binary.LittleEndian.Uint32(rest[8:]) != 0 {
		return nil, fmt.Errorf("unexpected damage-dealt template")
	}
	copy(rest[16:20], f32(mult))
	out := append([]byte{}, id...)
	out = append(out, coreStr(name)...)
	return append(out, rest...), nil
}

// buildToughness adds health / damage modifiers to every machine spawn definition.
func buildToughness(c MapConfig) ([]outFile, error) {
	if !c.Toughness.active() && !c.HumanTough.active() {
		return nil, nil
	}
	var td toughData
	if err := json.Unmarshal(readData("toughness.json"), &td); err != nil {
		return nil, err
	}
	dmType, _ := strconv.ParseUint(td.DMType, 16, 64)
	ddType, _ := strconv.ParseUint(td.DDType, 16, 64)
	type src struct{ data, path, folder string }
	var srcs []src
	for _, m := range allMachines {
		srcs = append(srcs, src{"orig/" + m + ".core", "entities/spawnsetups/robots/" + m + "/" + m + ".core", m})
	}
	for _, m := range td.DLC {
		srcs = append(srcs, src{"dlc/" + m + ".core", "entities/dlc1/spawnsetups/robots/" + m + "/" + m + ".core", m})
	}
	sort.Slice(srcs, func(i, j int) bool { return srcs[i].path < srcs[j].path })
	var out []outFile
	for _, s := range srcs {
		folder := s.folder
		objs, err := parseObjects(readData(s.data))
		if err != nil {
			return nil, fmt.Errorf("%s: %v", s.path, err)
		}
		nobjs, changed, err := applyTough(td, dmType, ddType, objs, s.path, func(name string) (int, int) {
			m := folder
			if m == "spraybot" && strings.Contains(name, "Cryo") {
				m = "spraybot_cryo"
			}
			g := c.groupOf(m)
			h := clampPct(pct(c.Toughness.Health, g) * pct(c.Toughness.HealthM, m) / 100)
			d := clampPct(pct(c.Toughness.Damage, g) * pct(c.Toughness.DamageM, m) / 100)
			return h, d
		})
		if err != nil {
			return nil, err
		}
		if changed {
			out = append(out, outFile{s.path, buildObjects(nobjs)})
		}
	}
	// human enemies: one file per human type
	if c.HumanTough.active() {
		keys := make([]string, 0, len(mapData.HTypes))
		for k := range mapData.HTypes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			t := mapData.HTypes[k]
			objs, err := parseObjects(readData("humans/" + k + ".core"))
			if err != nil {
				return nil, fmt.Errorf("%s: %v", t.Path, err)
			}
			h := clampPct(pct(c.HumanTough.Health, t.Faction) * pct(c.HumanTough.HealthM, k) / 100)
			d := clampPct(pct(c.HumanTough.Damage, t.Faction) * pct(c.HumanTough.DamageM, k) / 100)
			nobjs, changed, err := applyTough(td, dmType, ddType, objs, t.Path, func(string) (int, int) { return h, d })
			if err != nil {
				return nil, err
			}
			if changed {
				out = append(out, outFile{t.Path + ".core", buildObjects(nobjs)})
			}
		}
	}
	return out, nil
}

func clampPct(v int) int {
	if v < 10 {
		return 10
	}
	if v > 900 {
		return 900
	}
	return v
}

// applyTough adds health / damage modifiers to every spawn definition in a file.
// An existing damage-dealt modifier is scaled instead of adding a second one.
func applyTough(td toughData, dmType, ddType uint64, objs []object, path string, pick func(name string) (int, int)) ([]object, bool, error) {
	byID := map[string]int{}
	for i, o := range objs {
		byID[string(o.body[:16])] = i
	}
	scaled := map[int]bool{}
	var extra []object
	changed := false
	for i := range objs {
		if objs[i].typ != spawnSetupType {
			continue
		}
		b := objs[i].body
		f := spawnSetupFields(b)
		if strings.Contains(strings.ToLower(f.name), "corpse") {
			continue
		}
		h, d := pick(f.name)
		if h == 100 && d == 100 {
			continue
		}
		r := &reader{b, f.fact.end}
		cntAt := r.o
		n := int(r.u32())
		hasDD := -1
		for k := 0; k < n; k++ {
			rf := r.ref()
			if rf.kind == 1 {
				if j, ok := byID[string(rf.uuid)]; ok && objs[j].typ == ddType {
					hasDD = j
				}
			}
		}
		var add [][]byte
		tag := fmt.Sprintf("tough:%s:%x:", path, b[:16])
		if h != 100 {
			id := uuid5LE(tag + "health")
			body, err := buildDamageTaken(td, id, "EnemyRandomizer_Health", float32(100.0/float64(h)))
			if err != nil {
				return nil, false, err
			}
			extra = append(extra, object{dmType, body})
			add = append(add, id)
		}
		if d != 100 {
			if hasDD >= 0 {
				if !scaled[hasDD] {
					ob := objs[hasDD].body
					rr := &reader{ob, 16}
					rr.str()
					at := rr.o + 16
					if binary.LittleEndian.Uint32(ob[rr.o+8:]) == 0 && at+4 <= len(ob) {
						v := math.Float32frombits(binary.LittleEndian.Uint32(ob[at:]))
						nb := append([]byte{}, ob...)
						binary.LittleEndian.PutUint32(nb[at:], math.Float32bits(v*float32(d)/100))
						objs[hasDD].body = nb
						scaled[hasDD] = true
						changed = true
					}
				}
			} else {
				id := uuid5LE(tag + "damage")
				body, err := buildDamageDealt(td, id, "EnemyRandomizer_Damage", float32(float64(d)/100.0))
				if err != nil {
					return nil, false, err
				}
				extra = append(extra, object{ddType, body})
				add = append(add, id)
			}
		}
		if len(add) > 0 {
			nb := append([]byte{}, b[:r.o]...)
			for _, id := range add {
				nb = append(nb, 1)
				nb = append(nb, id...)
			}
			nb = append(nb, b[r.o:]...)
			binary.LittleEndian.PutUint32(nb[cntAt:], uint32(n+len(add)))
			objs[i].body = nb
			changed = true
		}
	}
	return append(objs, extra...), changed, nil
}
