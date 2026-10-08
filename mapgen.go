package main

import (
	"embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

//go:embed mapdata
var mapFS embed.FS

type Site struct {
	ID           int     `json:"id"`
	File         string  `json:"file"`
	Obj          int     `json:"obj"`
	Off          int     `json:"off"`
	Tile         [2]int  `json:"tile"`
	Machine      string  `json:"machine"`
	Variant      string  `json:"variant"`
	Kind         string  `json:"kind"`
	Scene        string  `json:"scene"`
	Cat          string  `json:"cat"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Exact        bool    `json:"exact"`
	Path         string  `json:"path"` // original spawnsetup path at this site
	DLC          bool    `json:"dlc"`
	CauldronName string  `json:"cauldron,omitempty"`
	Special      string  `json:"special,omitempty"` // boss / Hunting Ground spawnsetup name
}

type Human struct {
	ID       int     `json:"id"`
	File     string  `json:"file"`
	Obj      int     `json:"obj"`
	Off      int     `json:"off"`
	Kind     string  `json:"kind"`
	HType    string  `json:"htype"`
	Path     string  `json:"path"`
	UUID     string  `json:"uuid"`
	Scene    string  `json:"scene"`
	Cat      string  `json:"cat"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Exact    bool    `json:"exact"`
	DLC      bool    `json:"dlc"`
	Cauldron string  `json:"cauldron,omitempty"`
}

type HType struct {
	Faction string `json:"faction"` // bandit | eclipse | fwbandit
	Role    string `json:"role"`    // ranged | melee | heavy | sniper | special
	Path    string `json:"path"`
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
}

// HumanSetting: how human enemies are randomized
type HumanSetting struct {
	Mode       string `json:"mode"` // off | mix (types within their faction) | camps (whole encounters change faction) | chaos
	Seed       int64  `json:"seed"`
	Camps      bool   `json:"camps,omitempty"`       // also change the faction of bandit camps (camp activities)
	FWAnywhere bool   `json:"fw_anywhere,omitempty"` // Frozen Wilds bandits may appear outside The Cut
}

type Region struct {
	Name string     `json:"name"`
	Rect [4]float64 `json:"rect"`
}

type Cauldron struct {
	Name string  `json:"name"`
	Icon string  `json:"icon"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	DLC  bool    `json:"dlc"`
}

type MapData struct {
	Std       map[string]map[string][3]string `json:"std"` // machine -> variant -> [name, uuid, path]
	Sites     []Site                          `json:"sites"`
	Cauldrons []Cauldron                      `json:"cauldrons"`
	Ents      map[string]map[string][2]string `json:"ents"` // machine -> variant -> [entity uuid, entity path] (for herd lists)
	Regions   []Region                        `json:"regions"`
	Humans    []Human                         `json:"humans"`
	HTypes    map[string]HType                `json:"htypes"`
	Map       map[string]int                  `json:"map"`
}

type Area struct {
	Name        string       `json:"name"`
	Region      string       `json:"region,omitempty"`   // set when made from a ready-made region
	Disabled    bool         `json:"disabled,omitempty"` // toggled off: ignored, its sites fall through to the area underneath
	Color       string       `json:"color"`
	Tiles       [][2]int     `json:"tiles,omitempty"`
	Rects       [][4]float64 `json:"rects"` // world metres: x0, y0, x1, y1
	Mode        string       `json:"mode"`  // off | grouped | chaotic
	Seed        int64        `json:"seed"`
	Chance      int          `json:"chance"`                 // 0..100
	Difficulty  int          `json:"difficulty"`             // -2..2
	ClassChance int          `json:"class_chance,omitempty"` // 0..100: chance a normal spot gets a corrupted / Daemonic version
}

// WholeMap: one setting for every site on the map (areas are kept but ignored while it is on)
type WholeMap struct {
	On          bool   `json:"on"`
	Mode        string `json:"mode"` // grouped | chaotic
	Seed        int64  `json:"seed"`
	Chance      int    `json:"chance"`
	Difficulty  int    `json:"difficulty"`
	ClassChance int    `json:"class_chance,omitempty"`
	Herds       bool   `json:"herds"` // also roaming herds & patrols, with the same mode and seed
}

type HistoryEntry struct {
	Time    string          `json:"time"`
	Changes int             `json:"changes"`
	Summary string          `json:"summary"`
	Config  json.RawMessage `json:"config"`
}

// a copy of the settings without the saved presets, history and game folder (used for presets and history)
func (c MapConfig) snapshot() json.RawMessage {
	c.Presets, c.History, c.GamePath = nil, nil, ""
	b, _ := json.Marshal(c)
	return b
}

var weightOf = map[int]int{1: 1, 2: 3, 3: 8}

// pick one machine; plain uniform choice when every candidate has normal weight (keeps old seeds the same)
func (c MapConfig) pick(cands []string, rng *rand.Rand) string {
	return weightedPick(c.Weights, cands, rng)
}

func weightedPick(weights map[string]int, cands []string, rng *rand.Rand) string {
	ws := make([]int, len(cands))
	total, plain := 0, true
	for i, x := range cands {
		lv := weights[x]
		if lv == 0 {
			lv = 2
		}
		if lv != 2 {
			plain = false
		}
		ws[i] = weightOf[lv]
		total += ws[i]
	}
	if plain {
		return cands[rng.Intn(len(cands))]
	}
	r := rng.Intn(total)
	for i, w := range ws {
		if r < w {
			return cands[i]
		}
		r -= w
	}
	return cands[len(cands)-1]
}

// BossSetting: story bosses (Deathbringers, Corruptors, Frostclaws...) get their own option
type BossSetting struct {
	On   bool   `json:"on"`
	Mode string `json:"mode"` // big | medium | any
	Seed int64  `json:"seed"`
}

type HerdSetting struct {
	Mode string `json:"mode"` // off | grouped | chaotic
	Seed int64  `json:"seed"`
}

type MapConfig struct {
	Areas                     []Area              `json:"areas"`
	Rest                      Area                `json:"rest"`
	Herds                     HerdSetting         `json:"herds"`
	Whole                     WholeMap            `json:"whole"`
	Humans                    HumanSetting        `json:"humans"`
	Toughness                 Toughness           `json:"toughness,omitempty"`
	HumanTough                Toughness           `json:"human_tough,omitempty"` // keyed by faction (health/damage) and human type (health_m/damage_m)
	IncludeCauldrons          bool                `json:"include_cauldrons"`
	IncludeStory              bool                `json:"include_story"`
	IncludeHG                 bool                `json:"include_hg"` // Hunting Ground trial machines
	Bosses                    BossSetting         `json:"bosses"`
	Cross                     CrossSetting        `json:"cross"`
	Size                      int                 `json:"size,omitempty"`      // experimental: machine size in % (scales the spawn spot)
	FWAnywhereGrouped         bool                `json:"fw_anywhere_grouped"` // Frozen Wilds machines may appear outside The Cut in Grouped areas
	FWAnywhereChaotic         bool                `json:"fw_anywhere_chaotic"` // ...and in Chaotic areas
	ChaoticKeepFlyersSeparate bool                `json:"chaotic_keep_flyers_separate"`
	Groups                    map[string][]string `json:"groups"`
	GroupOrder                []string            `json:"group_order"`
	Excluded                  []string            `json:"excluded"`
	Banned                    []string            `json:"banned,omitempty"`    // switched off in the Machines list: replaced wherever they appear
	HerdSize                  int                 `json:"herd_size,omitempty"` // roaming herd size in percent (0 = 100)
	HWeights                  map[string]int      `json:"hweights,omitempty"`  // same for human types
	Pins                      map[string]string   `json:"pins,omitempty"`      // site id -> machine chosen by hand, or "keep"
	planOnly                  bool
	tally                     map[string]int             // herd machine counts, filled when planOnly
	Weights                   map[string]int             `json:"weights,omitempty"` // how often a machine is picked: 1 rare, 2 normal (default), 3 common
	Presets                   map[string]json.RawMessage `json:"presets,omitempty"` // saved setups, by name
	History                   []HistoryEntry             `json:"history,omitempty"` // last builds (newest first)
	GamePath                  string                     `json:"game_path"`
}

func defaultMapConfig() MapConfig {
	d := defaultSettings()
	return MapConfig{
		Rest:                      Area{Name: "Rest of the world", Color: "#8a8f98", Mode: "off", Seed: 1, Chance: 100},
		Herds:                     HerdSetting{Mode: "off", Seed: 1},
		Whole:                     WholeMap{Mode: "grouped", Seed: 1, Chance: 100, Herds: true},
		ChaoticKeepFlyersSeparate: true,
		Groups:                    d.Groups, GroupOrder: d.GroupOrder, Excluded: d.Excluded,
	}
}

// fill in machines added in newer versions (Frozen Wilds) that an older settings file doesn't mention
func (c *MapConfig) migrate() {
	if c.Whole.Mode == "" { // settings saved before the whole-map option existed
		c.Whole = WholeMap{Mode: "grouped", Seed: 1, Chance: 100, Herds: true}
	}
	known := map[string]bool{}
	for _, l := range c.Groups {
		for _, m := range l {
			known[m] = true
		}
	}
	for _, m := range c.Excluded {
		known[m] = true
	}
	d := defaultSettings()
	if c.Groups == nil {
		c.Groups = map[string][]string{}
	}
	for _, g := range d.GroupOrder {
		for _, m := range d.Groups[g] {
			if !known[m] {
				c.Groups[g] = append(c.Groups[g], m)
			}
		}
	}
	if len(c.GroupOrder) == 0 {
		c.GroupOrder = d.GroupOrder
	}
}

var mapData MapData

func loadMapData() {
	b, err := mapFS.ReadFile("mapdata/sites.json")
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, &mapData); err != nil {
		panic(err)
	}
}

func sceneBytes(file string) []byte {
	b, err := readScene(strings.ReplaceAll(file, "/", "__"))
	if err != nil {
		panic(err)
	}
	return b
}

var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// Decima string checksum: CRC32-C, zero initial value, no final xor, top bit cleared
func decimaCRC(s string) uint32 {
	return (^crc32.Update(0xffffffff, castagnoli, []byte(s))) & 0x7fffffff
}

// variants a donor may stand in with, best first. Corrupted sites only take corrupted donors,
// so a corrupted zone never gets a machine from a different faction.
var variantFallback = map[string][]string{
	"normal":    {"normal", "dlc", "cc"},
	"dlc":       {"dlc", "normal", "cc"},
	"cc":        {"cc", "dlc", "normal"},
	"corrupted": {"corrupted"},
}

func donorVariant(machine, variant string) ([3]string, bool) {
	std := mapData.Std[machine]
	for _, v := range variantFallback[variant] {
		if d, ok := std[v]; ok {
			return d, true
		}
	}
	return [3]string{}, false
}

func spawnRef(machine, variant string) ([]byte, error) {
	v, ok := donorVariant(machine, variant)
	if !ok {
		return nil, fmt.Errorf("no %s spawn definition for %s", variant, machine)
	}
	u, err := parseUUIDLE(v[1])
	if err != nil {
		return nil, err
	}
	path := v[2]
	out := []byte{2}
	out = append(out, u...)
	var t [8]byte
	binary.LittleEndian.PutUint32(t[0:], uint32(len(path)))
	binary.LittleEndian.PutUint32(t[4:], decimaCRC(path))
	out = append(out, t[:]...)
	out = append(out, path...)
	return out, nil
}

func parseUUIDLE(s string) ([]byte, error) {
	h := strings.ReplaceAll(s, "-", "")
	if len(h) != 32 {
		return nil, fmt.Errorf("bad uuid %s", s)
	}
	var b [16]byte
	for i := 0; i < 16; i++ {
		fmt.Sscanf(h[2*i:2*i+2], "%02x", &b[i])
	}
	return []byte{b[3], b[2], b[1], b[0], b[5], b[4], b[7], b[6], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15]}, nil
}

func (c MapConfig) tier() map[string]int {
	t := map[string]int{}
	for i, g := range []string{"Small", "Medium", "Large"} {
		for _, m := range c.Groups[g] {
			t[m] = i + 1
		}
	}
	for _, m := range c.Groups["Flyer"] {
		t[m] = 9
	}
	return t
}

func (c MapConfig) wholeArea() Area {
	w := c.Whole
	if w.Mode != "chaotic" {
		w.Mode = "grouped"
	}
	return Area{Name: "Whole map", Mode: w.Mode, Seed: w.Seed, Chance: w.Chance, Difficulty: w.Difficulty, ClassChance: w.ClassChance}
}

// effective herd setting (the whole-map setting takes over when it includes herds)
func (c MapConfig) herdSetting() HerdSetting {
	if c.Whole.On && c.Whole.Herds {
		return HerdSetting{Mode: c.wholeArea().Mode, Seed: c.Whole.Seed}
	}
	return c.Herds
}

func (c MapConfig) areaFor(s Site) Area {
	if c.Whole.On {
		return c.wholeArea()
	}
	// later areas win where boxes overlap
	for i := len(c.Areas) - 1; i >= 0; i-- {
		if c.Areas[i].Disabled {
			continue
		}
		for _, r := range c.Areas[i].Rects {
			x0, x1 := r[0], r[2]
			if x0 > x1 {
				x0, x1 = x1, x0
			}
			y0, y1 := r[1], r[3]
			if y0 > y1 {
				y0, y1 = y1, y0
			}
			if s.X >= x0 && s.X <= x1 && s.Y >= y0 && s.Y <= y1 {
				return c.Areas[i]
			}
		}
	}
	if c.Rest.Disabled {
		return Area{Mode: "off"}
	}
	return c.Rest
}

func siteRng(seed int64, id int) *rand.Rand {
	return rand.New(rand.NewSource(seed*1000003 + int64(id)*7919 + 17))
}

var frozenWilds = map[string]bool{"hellhound": true, "cryobear": true, "infernobear": true}

// may machine x stand in at a site? (Frozen Wilds machines stay in The Cut unless the area's mode has its tick box on)
func (c MapConfig) fwAllowed(x, mode string, dlcSite bool) bool {
	if !frozenWilds[x] || dlcSite {
		return true
	}
	return (mode == "grouped" && c.FWAnywhereGrouped) || (mode == "chaotic" && c.FWAnywhereChaotic)
}

func (c MapConfig) isBanned(m string) bool {
	for _, b := range c.Banned {
		if b == m {
			return true
		}
	}
	return false
}

// replacement for a switched-off machine: same size group if possible, otherwise any allowed machine
func pickReplacement(c MapConfig, m, variant string, dlcSite bool, mode string, rng *rand.Rand, ok func(string) bool) string {
	ex := map[string]bool{}
	for _, e := range c.Excluded {
		ex[e] = true
	}
	tier := c.tier()
	t, known := tier[m]
	if !known {
		t = 2 // never-changed machines (e.g. Corruptor, Deathbringer) are treated as Medium
		if m == "warrobot" {
			t = 3
		}
	}
	var same, any []string
	for _, g := range c.GroupOrder {
		for _, x := range c.Groups[g] {
			if x == m || ex[x] || c.isBanned(x) || !c.fwAllowed(x, mode, dlcSite) || (ok != nil && !ok(x)) {
				continue
			}
			if _, has := donorVariant(x, variant); !has {
				continue
			}
			any = append(any, x)
			if tier[x] == t {
				same = append(same, x)
			}
		}
	}
	sort.Strings(same)
	sort.Strings(any)
	if len(same) > 0 {
		return c.pick(same, rng)
	}
	if len(any) > 0 {
		return c.pick(any, rng)
	}
	return m
}

// pickBoss: a boss becomes another big machine ("big") or anything at all ("any").
// Switched-off and never-randomized machines are not used; Frozen Wilds machines follow the FW ticks.
func pickBoss(c MapConfig, s Site, rng *rand.Rand) string {
	ex := map[string]bool{}
	for _, e := range c.Excluded {
		ex[e] = true
	}
	tier := c.tier()
	var cands []string
	for _, g := range c.GroupOrder {
		for _, x := range c.Groups[g] {
			if x == s.Machine || ex[x] || c.isBanned(x) || !c.fwAllowed(x, "grouped", s.DLC) {
				continue
			}
			if _, has := donorVariant(x, s.Variant); !has {
				continue
			}
			switch c.Bosses.Mode {
			case "any":
			case "medium": // Medium or Large, no flyers
				if tier[x] != 2 && tier[x] != 3 {
					continue
				}
			default: // big
				if tier[x] != 3 {
					continue
				}
			}
			cands = append(cands, x)
		}
	}
	sort.Strings(cands)
	if len(cands) == 0 {
		return s.Machine
	}
	return c.pick(cands, rng)
}

func pickDonor(c MapConfig, a Area, m, variant string, dlcSite bool, rng *rand.Rand, ok func(string) bool) string {
	if c.isBanned(m) {
		mode := a.Mode
		if mode != "chaotic" {
			mode = "grouped"
		}
		return pickReplacement(c, m, variant, dlcSite, mode, rng, ok)
	}
	ex := map[string]bool{}
	for _, e := range c.Excluded {
		ex[e] = true
	}
	for _, b := range c.Banned {
		ex[b] = true
	}
	if ex[m] || a.Mode == "off" || a.Mode == "" {
		return m
	}
	if rng.Intn(100) >= a.Chance {
		return m
	}
	tier := c.tier()
	t, known := tier[m]
	if !known {
		return m
	}
	var cands []string
	all := []string{}
	for _, g := range c.GroupOrder {
		for _, x := range c.Groups[g] {
			if _, has := donorVariant(x, variant); has && !ex[x] && c.fwAllowed(x, a.Mode, dlcSite) && (ok == nil || ok(x)) {
				all = append(all, x)
			}
		}
	}
	sort.Strings(all)
	if t == 9 { // flyer
		for _, x := range all {
			if x != m && (tier[x] == 9 || (a.Mode == "chaotic" && !c.ChaoticKeepFlyersSeparate)) {
				cands = append(cands, x)
			}
		}
	} else if a.Mode == "grouped" {
		want := t + a.Difficulty
		if want < 1 {
			want = 1
		}
		if want > 3 {
			want = 3
		}
		for _, x := range all {
			if tier[x] == want && x != m {
				cands = append(cands, x)
			}
		}
	} else { // chaotic
		for _, x := range all {
			tx := tier[x]
			if x == m || (tx == 9 && c.ChaoticKeepFlyersSeparate) {
				continue
			}
			if tx == 9 {
				tx = 2
			}
			if a.Difficulty > 0 && tx < t {
				continue
			}
			if a.Difficulty < 0 && tx > t {
				continue
			}
			if a.Difficulty >= 2 && tx <= t && t < 3 {
				continue
			}
			if a.Difficulty <= -2 && tx >= t && t > 1 {
				continue
			}
			cands = append(cands, x)
		}
	}
	if len(cands) == 0 {
		return m
	}
	return c.pick(cands, rng)
}

type Change struct {
	Human   bool   `json:"human,omitempty"`
	ID      int    `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Variant string `json:"variant"`         // the version placed there (normal, corrupted, dlc, cc = Daemonic)
	Cross   bool   `json:"cross,omitempty"` // machine spot -> human (To = human type), or human spot -> machine (Human set, To = machine)
}

// classRoll: with the area's Corrupted/Daemonic chance, a normal spot gets the corrupted version
// (or the Daemonic version inside The Cut) of whatever machine ends up there.
func classRoll(c MapConfig, a Area, s Site, donor string) string {
	if a.ClassChance <= 0 || a.Mode == "off" || a.Mode == "" || (s.Variant != "normal" && s.Variant != "dlc") {
		return s.Variant
	}
	for _, e := range c.Excluded {
		if e == donor {
			return s.Variant
		}
	}
	if siteRng(a.Seed+7919, s.ID).Intn(100) >= a.ClassChance {
		return s.Variant
	}
	want := "corrupted"
	if s.DLC {
		want = "cc"
	}
	if _, ok := mapData.Std[donor][want]; ok {
		return want
	}
	return s.Variant
}

func buildMapPatch(c MapConfig) ([]byte, []Change, error) {
	changes := []Change{}
	bySite := map[string][]Site{}
	donorOf := map[int]string{}
	variantOf := map[int]string{}
	type pick struct {
		s    Site
		a    Area
		d, v string
	}
	var picks []pick
	crossH, crossM := planCross(c)
	for _, s := range mapData.Sites {
		if pin, ok := c.Pins[strconv.Itoa(s.ID)]; ok { // picked by hand on the map
			if pin == "keep" || pin == s.Machine {
				continue
			}
			if _, has := mapData.Std[pin]; has {
				v := s.Variant
				if _, ok := donorVariant(pin, v); !ok {
					v = "normal"
				}
				picks = append(picks, pick{s, Area{Name: "Pinned", Mode: "off"}, pin, v})
				continue
			}
		}
		if _, crossed := crossH[s.ID]; crossed {
			continue
		}
		if s.Cat == "cauldron" && !c.IncludeCauldrons {
			continue
		}
		if s.Cat == "story" && !c.IncludeStory {
			continue
		}
		if s.Cat == "hg" && !c.IncludeHG {
			continue
		}
		if s.Cat == "boss" {
			if !c.Bosses.On {
				continue
			}
			a := Area{Name: "Bosses", Mode: "grouped", Seed: c.Bosses.Seed + 4241, Chance: 100}
			picks = append(picks, pick{s, a, pickBoss(c, s, siteRng(a.Seed, s.ID)), ""})
			continue
		}
		a := c.areaFor(s)
		if s.Cat == "roaming" { // convoys and random encounters follow the roaming herds setting
			hs := c.herdSetting()
			a = Area{Name: "Roaming", Mode: hs.Mode, Seed: hs.Seed + 977, Chance: 100}
			if c.Whole.On && c.Whole.Herds {
				a.Chance, a.Difficulty, a.ClassChance = c.Whole.Chance, c.Whole.Difficulty, c.Whole.ClassChance
			}
		}
		d := pickDonor(c, a, s.Machine, s.Variant, s.DLC, siteRng(a.Seed, s.ID), nil)
		picks = append(picks, pick{s, a, d, ""})
	}
	// room check: a Large machine only goes on a single spot with nothing else close by,
	// otherwise big machines spawn inside each other
	tier := c.tier()
	final := map[int]string{}
	for _, p := range picks {
		final[p.s.ID] = p.d
	}
	var bigAt [][3]float64 // placed big machines: x, y, file index
	fileIdx := map[string]float64{}
	for i := range picks {
		p := &picks[i]
		if p.d != p.s.Machine && tier[p.d] == 3 && tier[p.s.Machine] != 3 && p.s.Kind != "AIBehaviorGroupMember" && p.s.Cat != "boss" && p.a.Name != "Pinned" {
			x, y, ok := spotPos(p.s)
			fi, seen := fileIdx[p.s.File]
			if !seen {
				fi = float64(len(fileIdx))
				fileIdx[p.s.File] = fi
			}
			crowded := !ok
			if ok {
				for _, o := range mapData.Sites {
					if o.File != p.s.File || o.ID == p.s.ID || o.Kind == "AIBehaviorGroupMember" {
						continue
					}
					if ox, oy, ok2 := spotPos(o); ok2 && (ox-x)*(ox-x)+(oy-y)*(oy-y) < 12*12 {
						crowded = true
						break
					}
				}
				for _, b := range bigAt {
					if b[2] == fi && (b[0]-x)*(b[0]-x)+(b[1]-y)*(b[1]-y) < 30*30 {
						crowded = true
						break
					}
				}
			}
			if crowded {
				notLarge := func(m string) bool { return tier[m] != 3 }
				p.d = pickDonor(c, p.a, p.s.Machine, p.s.Variant, p.s.DLC, siteRng(p.a.Seed+31, p.s.ID), notLarge)
				if tier[p.d] == 3 && p.d != p.s.Machine {
					p.d = p.s.Machine
				}
				final[p.s.ID] = p.d
			} else {
				bigAt = append(bigAt, [3]float64{x, y, fi})
			}
		}
	}
	for _, p := range picks {
		s, d := p.s, p.d
		v := p.v
		if v == "" {
			v = classRoll(c, p.a, s, d)
		}
		if d != s.Machine || v != s.Variant {
			donorOf[s.ID] = d
			variantOf[s.ID] = v
			bySite[s.File] = append(bySite[s.File], s)
			changes = append(changes, Change{ID: s.ID, From: s.Machine, To: d, Variant: resolvedVariant(d, v)})
		}
	}
	// humans <-> machines (experimental)
	var crossSites []Site
	for _, s := range mapData.Sites {
		if ht, ok := crossH[s.ID]; ok {
			crossSites = append(crossSites, s)
			changes = append(changes, Change{ID: s.ID, From: s.Machine, To: ht, Cross: true})
		}
	}
	hch, hplan := planHumans(c)
	for _, ch := range hch {
		if _, ok := crossM[ch.ID]; ok {
			delete(hplan, ch.ID)
			continue
		}
		changes = append(changes, ch)
	}
	for _, h := range mapData.Humans {
		if m, ok := crossM[h.ID]; ok {
			changes = append(changes, Change{Human: true, Cross: true, ID: h.ID, From: h.HType, To: m, Variant: "normal"})
		}
	}
	if c.planOnly { // preview: what would be built, without making the patch
		if _, err := buildHerds(c); err != nil {
			return nil, nil, err
		}
		return nil, changes, nil
	}
	type edit struct {
		obj, off      int
		oldPath, what string
		newRef        []byte
		after         func(nb []byte, at int)
	}
	edits := map[string][]edit{}
	for f, ss := range bySite {
		for _, s := range ss {
			s := s
			nr, err := spawnRef(donorOf[s.ID], variantOf[s.ID])
			if err != nil {
				return nil, nil, err
			}
			e := edit{obj: s.Obj, off: s.Off, oldPath: s.Path, what: fmt.Sprintf("site %d", s.ID), newRef: nr}
			if s.Kind == "AIBehaviorGroupMember" {
				e.after = func(nb []byte, at int) { groupCount(c, nb, at, s.Machine, donorOf[s.ID]) }
			}
			edits[f] = append(edits[f], e)
		}
	}
	for _, s := range crossSites {
		t := mapData.HTypes[crossH[s.ID]]
		nr, err := entRef(t.UUID, t.Path)
		if err != nil {
			return nil, nil, err
		}
		nr[0] = 2
		edits[s.File] = append(edits[s.File], edit{obj: s.Obj, off: s.Off, oldPath: s.Path, what: fmt.Sprintf("site %d (to human)", s.ID), newRef: nr})
	}
	for _, h := range mapData.Humans {
		m, ok := crossM[h.ID]
		if !ok {
			continue
		}
		v := "normal"
		if h.DLC {
			v = "dlc"
		}
		nr, err := spawnRef(m, v)
		if err != nil {
			return nil, nil, err
		}
		edits[h.File] = append(edits[h.File], edit{obj: h.Obj, off: h.Off, oldPath: h.Path, what: fmt.Sprintf("human %d (to machine)", h.ID), newRef: nr})
	}
	for _, h := range mapData.Humans {
		nt, ok := hplan[h.ID]
		if !ok {
			continue
		}
		t := mapData.HTypes[nt]
		nr, err := entRef(t.UUID, t.Path)
		if err != nil {
			return nil, nil, err
		}
		nr[0] = 2
		edits[h.File] = append(edits[h.File], edit{obj: h.Obj, off: h.Off, oldPath: h.Path, what: fmt.Sprintf("human %d", h.ID), newRef: nr})
	}
	var files []outFile
	fileNames := make([]string, 0, len(edits))
	for f := range edits {
		fileNames = append(fileNames, f)
	}
	sort.Strings(fileNames)
	for _, f := range fileNames {
		objs, err := parseObjects(sceneBytes(f))
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %v", f, err)
		}
		es := edits[f]
		sort.Slice(es, func(i, j int) bool {
			if es[i].obj != es[j].obj {
				return es[i].obj < es[j].obj
			}
			return es[i].off > es[j].off // edit from the back of each object
		})
		for _, e := range es {
			body := objs[e.obj].body
			if e.off+25+len(e.oldPath) > len(body) || body[e.off] != 2 || string(body[e.off+25:e.off+25+len(e.oldPath)]) != e.oldPath {
				return nil, nil, fmt.Errorf("unexpected data at %s in %s", e.what, f)
			}
			nb := append([]byte{}, body[:e.off]...)
			nb = append(nb, e.newRef...)
			nb = append(nb, body[e.off+25+len(e.oldPath):]...)
			if e.after != nil {
				e.after(nb, e.off+len(e.newRef))
			}
			objs[e.obj].body = nb
		}
		files = append(files, outFile{f, buildObjects(objs)})
	}
	// roaming herds and patrols: every herd member picks its own machine, so herds come out mixed
	if h := c.herdSetting(); h.Mode == "grouped" || h.Mode == "chaotic" || len(c.Banned) > 0 || (c.HerdSize > 0 && c.HerdSize != 100) {
		hf, err := buildHerds(c)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, hf...)
	}
	tf, err := buildToughness(c)
	if err != nil {
		return nil, nil, err
	}
	files = append(files, tf...)
	out, err := generateParts(Settings{Seed: c.herdSetting().Seed}, map[string]string{}, false, false, files)
	if err != nil {
		return nil, nil, err
	}
	return packRaw(out), changes, nil
}

func entRef(u, path string) ([]byte, error) {
	ub, err := parseUUIDLE(u)
	if err != nil {
		return nil, err
	}
	out := []byte{3}
	out = append(out, ub...)
	var t [8]byte
	binary.LittleEndian.PutUint32(t[0:], uint32(len(path)))
	binary.LittleEndian.PutUint32(t[4:], decimaCRC(path))
	out = append(out, t[:]...)
	return append(out, path...), nil
}

// herd lists: each EntityGroupMemberResource holds one machine entity reference (kind 3)
func buildHerds(c MapConfig) ([]outFile, error) {
	type mv struct{ m, v string }
	byEnt := map[string]mv{}
	for m, vs := range mapData.Ents {
		for v, e := range vs {
			if v != "normal" && v != "corrupted" {
				continue
			}
			ub, err := parseUUIDLE(e[0])
			if err != nil {
				return nil, err
			}
			byEnt[string(ub)+e[1]] = mv{m, v}
		}
	}
	hs := c.herdSetting()
	a := Area{Mode: hs.Mode, Seed: hs.Seed, Chance: 100}
	if c.Whole.On && c.Whole.Herds {
		a.Chance, a.Difficulty = c.Whole.Chance, c.Whole.Difficulty
	}
	hasEnt := func(v string) func(string) bool {
		return func(x string) bool { _, ok := mapData.Ents[x][v]; return ok }
	}
	var out []outFile
	member := 0
	for hi, h := range []string{"resourcegatheringherd", "recongroup"} {
		objs, err := parseObjects(readData("orig/herd_" + h + ".core"))
		if err != nil {
			return nil, err
		}
		for oi := range objs {
			d := objs[oi].body
			var res []byte
			for i := 0; i < len(d); {
				if d[i] == 3 && i+25 <= len(d) {
					l := int(binary.LittleEndian.Uint32(d[i+17:]))
					if l > 0 && l < 200 && i+25+l <= len(d) {
						if k, ok := byEnt[string(d[i+1:i+17])+string(d[i+25:i+25+l])]; ok {
							member++
							rng := siteRng(hs.Seed, 100000*(hi+1)+member)
							donor := k.m
							if k.m == "warrobot" && !c.Bosses.On {
								// the patrol list's own Deathbringer belongs to a story fight: it follows the Bosses option
							} else if a.Mode == "grouped" || a.Mode == "chaotic" || c.isBanned(k.m) {
								donor = pickDonor(c, a, k.m, k.v, false, rng, hasEnt(k.v))
							}
							e := mapData.Ents[donor][k.v]
							r, err := entRef(e[0], e[1])
							if err != nil {
								return nil, err
							}
							res = append(res, r...)
							i += 25 + l
							// the member's machine count follows its entity reference
							if i+4 <= len(d) {
								n := int(binary.LittleEndian.Uint32(d[i:]))
								var cb [4]byte
								hn := herdCount(c, n, k.m, donor)
								if c.tally != nil {
									c.tally[donor] += hn
								}
								binary.LittleEndian.PutUint32(cb[:], uint32(hn))
								res = append(res, cb[:]...)
								i += 4
							}
							continue
						}
					}
				}
				res = append(res, d[i])
				i++
			}
			objs[oi].body = res
		}
		out = append(out, outFile{"entities/herds/" + h + ".core", buildObjects(objs)})
	}
	return out, nil
}

// the version actually used after falling back (e.g. a Grazer at a Daemonic spot is a normal Grazer)
func resolvedVariant(machine, variant string) string {
	for _, v := range variantFallback[variant] {
		if _, ok := mapData.Std[machine][v]; ok {
			return v
		}
	}
	return variant
}

// herdCount scales a herd member's machine count by the herd size setting. Big machines
// (the Large group) that take over a member are capped so a big Grazer herd can't become
// a big Thunderjaw herd.
func herdCount(c MapConfig, n int, orig, donor string) int {
	if n <= 0 || n > 64 {
		return n // not a count we understand - leave it alone
	}
	size := c.HerdSize
	if size <= 0 {
		size = 100
	}
	nn := (n*size + 50) / 100
	if nn < 1 {
		nn = 1
	}
	if nn > 40 {
		nn = 40
	}
	// a bigger machine taking over a member of smaller machines gets a capped count
	tier := c.tier()
	caps := map[int]int{2: 6, 3: 1, 9: 3} // Medium, Large, Flyer
	rank := func(t int) int {
		if t == 9 {
			return 2
		}
		return t
	}
	if lim, ok := caps[tier[donor]]; ok && donor != orig && rank(tier[donor]) > rank(tier[orig]) || (tier[donor] == 9 && tier[orig] != 9) {
		if !ok {
			lim = 3
		}
		limit := (lim*size + 50) / 100
		if limit < 1 {
			limit = 1
		}
		if nn > limit {
			nn = limit
		}
	}
	return nn
}

// position of a single spawn spot inside its scene (the object starts with its transform)
var posCache = map[int][3]float64{}

func spotPos(s Site) (float64, float64, bool) {
	if p, ok := posCache[s.ID]; ok {
		return p[0], p[1], p[2] == 1
	}
	res := [3]float64{}
	if objs, err := parseObjects(sceneBytes(s.File)); err == nil && s.Obj < len(objs) && len(objs[s.Obj].body) >= 24 {
		b := objs[s.Obj].body
		x := math.Float64frombits(binary.LittleEndian.Uint64(b[0:]))
		y := math.Float64frombits(binary.LittleEndian.Uint64(b[8:]))
		if !math.IsNaN(x) && !math.IsNaN(y) && math.Abs(x) < 5000 && math.Abs(y) < 5000 {
			res = [3]float64{x, y, 1}
		}
	}
	posCache[s.ID] = res
	return res[0], res[1], res[2] == 1
}

// groupCount: a group spawn (min count, max count, radius...) taken over by a bigger machine
// spawns fewer of them so they don't end up inside each other.
func groupCount(c MapConfig, b []byte, at int, orig, donor string) {
	if at+8 > len(b) {
		return
	}
	mn := int(binary.LittleEndian.Uint32(b[at:]))
	mx := int(binary.LittleEndian.Uint32(b[at+4:]))
	if mn <= 0 || mx < mn || mx > 64 {
		return
	}
	tier := c.tier()
	rank := func(t int) int {
		if t == 9 {
			return 2
		}
		return t
	}
	td, to := tier[donor], tier[orig]
	limit := 0
	switch {
	case td == 3 && to != 3:
		limit = 1
	case td == 9 && to != 9:
		limit = 2
	case rank(td) > rank(to):
		limit = 3
	}
	if limit == 0 || mx <= limit {
		return
	}
	if mn > limit {
		mn = limit
	}
	binary.LittleEndian.PutUint32(b[at:], uint32(mn))
	binary.LittleEndian.PutUint32(b[at+4:], uint32(limit))
}

// planHumans decides the new type of each human enemy spawn.
func planHumans(c MapConfig) ([]Change, map[int]string) {
	hs := c.Humans
	plan := map[int]string{}
	var ch []Change
	if hs.Mode == "" || hs.Mode == "off" || len(mapData.Humans) == 0 {
		return ch, plan
	}
	keys := make([]string, 0, len(mapData.HTypes))
	for k := range mapData.HTypes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	allowed := func(k string, dlcSpot bool) bool {
		return mapData.HTypes[k].Faction != "fwbandit" || dlcSpot || hs.FWAnywhere
	}
	of := func(faction, role string, dlcSpot bool) []string {
		var out []string
		for _, k := range keys {
			t := mapData.HTypes[k]
			if (faction == "" || t.Faction == faction) && (role == "" || t.Role == role) && allowed(k, dlcSpot) {
				out = append(out, k)
			}
		}
		return out
	}
	// in "camps" mode every encounter (scene file) gets one new faction, so a camp stays on one side
	target := map[string]string{}
	if hs.Mode == "camps" {
		files := map[string][]Human{}
		for _, h := range mapData.Humans {
			files[h.File] = append(files[h.File], h)
		}
		names := make([]string, 0, len(files))
		for f := range files {
			names = append(names, f)
		}
		sort.Strings(names)
		for i, f := range names {
			hh := files[f]
			if hh[0].Cat == "camp" && !hs.Camps {
				continue
			}
			own := mapData.HTypes[hh[0].HType].Faction
			var opts []string
			for _, fac := range []string{"bandit", "eclipse", "fwbandit"} {
				if fac != own && (fac != "fwbandit" || hh[0].DLC || hs.FWAnywhere) {
					opts = append(opts, fac)
				}
			}
			if len(opts) > 0 {
				target[f] = opts[siteRng(hs.Seed+4243, i).Intn(len(opts))]
			}
		}
	}
	for _, h := range mapData.Humans {
		if (h.Cat == "story" && !c.IncludeStory) || (h.Cat == "cauldron" && !c.IncludeCauldrons) {
			continue
		}
		cur := mapData.HTypes[h.HType]
		rng := siteRng(hs.Seed+5003, h.ID)
		var cands []string
		switch hs.Mode {
		case "mix":
			cands = of(cur.Faction, "", h.DLC)
		case "camps":
			if fac, ok := target[h.File]; ok {
				cands = of(fac, cur.Role, h.DLC)
				if len(cands) == 0 {
					cands = of(fac, "", h.DLC)
				}
			} else {
				cands = of(cur.Faction, "", h.DLC) // bandit camps without the camp tick: mix within the faction
			}
		case "chaos":
			cands = of("", "", h.DLC)
		}
		if len(cands) == 0 {
			continue
		}
		nt := weightedPick(c.HWeights, cands, rng)
		if nt != h.HType {
			plan[h.ID] = nt
			ch = append(ch, Change{Human: true, ID: h.ID, From: h.HType, To: nt})
		}
	}
	return ch, plan
}

// scene (level) files come from the player's game (see gamedata.go)
var sceneCache = map[string][]byte{}

func readScene(name string) ([]byte, error) {
	return gameFile(strings.ReplaceAll(name, "__", "/"))
}
