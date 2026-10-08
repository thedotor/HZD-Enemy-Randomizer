package main

import (
	"math/rand"
	"sort"
	"strconv"
)

// CrossSetting: some machine spots become human enemies and some human spots become machines (experimental)
type CrossSetting struct {
	On        bool  `json:"on"`
	ToHuman   int   `json:"to_human"`   // % of machine spots that become humans
	ToMachine int   `json:"to_machine"` // % of human spots that become machines
	Seed      int64 `json:"seed"`
	Camps     bool  `json:"camps,omitempty"` // bandit camps may get machines too
}

func crossKind(k string) bool {
	return k == "0xd9a67b7355712bb5" || k == "Spawnpoint" || k == "MultiSpawnpoint"
}

// planCross picks which machine spots become humans (site id -> human type)
// and which human spots become machines (human id -> machine).
func planCross(c MapConfig) (map[int]string, map[int]string) {
	toH, toM := map[int]string{}, map[int]string{}
	x := c.Cross
	if !x.On || (x.ToHuman <= 0 && x.ToMachine <= 0) {
		return toH, toM
	}
	ex := map[string]bool{}
	for _, e := range c.Excluded {
		ex[e] = true
	}
	tier := c.tier()
	keys := make([]string, 0, len(mapData.HTypes))
	for k := range mapData.HTypes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	of := func(faction string, roles ...string) []string {
		var out []string
		for _, k := range keys {
			t := mapData.HTypes[k]
			if t.Faction != faction {
				continue
			}
			for _, r := range roles {
				if t.Role == r {
					out = append(out, k)
					break
				}
			}
		}
		return out
	}
	catOK := func(cat string) bool {
		switch cat {
		case "world":
			return true
		case "story":
			return c.IncludeStory
		case "cauldron":
			return c.IncludeCauldrons
		case "hg":
			return c.IncludeHG
		}
		return false
	}
	// machine spots -> humans
	for _, s := range mapData.Sites {
		if x.ToHuman <= 0 || !crossKind(s.Kind) || !catOK(s.Cat) || ex[s.Machine] {
			continue
		}
		if _, pinned := c.Pins[strconv.Itoa(s.ID)]; pinned {
			continue
		}
		rng := siteRng(x.Seed+611, s.ID)
		if rng.Intn(100) >= x.ToHuman {
			continue
		}
		fac := "fwbandit"
		if !s.DLC {
			fs := []string{"bandit", "eclipse"}
			if c.Humans.FWAnywhere {
				fs = append(fs, "fwbandit")
			}
			fac = fs[rng.Intn(len(fs))]
		}
		var roles []string
		switch tier[s.Machine] {
		case 1:
			roles = []string{"ranged", "melee"}
		case 3:
			roles = []string{"heavy"}
		case 9:
			roles = []string{"sniper", "ranged"}
		default:
			roles = []string{"ranged", "melee", "special"}
		}
		cands := of(fac, roles...)
		if len(cands) == 0 {
			cands = of(fac, "ranged", "melee", "heavy", "sniper", "special")
		}
		if len(cands) > 0 {
			toH[s.ID] = weightedPick(c.HWeights, cands, rng)
		}
	}
	// human spots -> machines
	byTier := func(t int, dlc bool) []string {
		var out []string
		for _, g := range c.GroupOrder {
			for _, m := range c.Groups[g] {
				if tier[m] == t && !ex[m] && !c.isBanned(m) && c.fwAllowed(m, "grouped", dlc) {
					if _, ok := donorVariant(m, "normal"); ok {
						out = append(out, m)
					}
				}
			}
		}
		sort.Strings(out)
		return out
	}
	near := func(h Human) bool {
		for _, o := range mapData.Humans {
			if o.ID != h.ID && o.File == h.File && (o.X-h.X)*(o.X-h.X)+(o.Y-h.Y)*(o.Y-h.Y) < 12*12 {
				return true
			}
		}
		return false
	}
	for _, h := range mapData.Humans {
		if x.ToMachine <= 0 || !crossKind(h.Kind) {
			continue
		}
		if !(catOK(h.Cat) || (h.Cat == "camp" && x.Camps)) {
			continue
		}
		rng := siteRng(x.Seed+733, h.ID)
		if rng.Intn(100) >= x.ToMachine {
			continue
		}
		t := mapData.HTypes[h.HType]
		want := 2
		if t.Role == "heavy" && !near(h) {
			want = 3
		} else if (t.Role == "ranged" || t.Role == "melee") && rng.Intn(2) == 0 {
			want = 1
		}
		cands := byTier(want, h.DLC)
		if len(cands) == 0 {
			cands = byTier(2, h.DLC)
		}
		if len(cands) > 0 {
			toM[h.ID] = c.pick(cands, rng)
		}
	}
	return toH, toM
}

var _ = rand.Intn
