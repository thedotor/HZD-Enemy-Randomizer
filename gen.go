package main

import (
	"bytes"
	"crypto/sha1"
	"embed"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

//go:embed data
var dataFS embed.FS

const spawnSetupType = 0xd3725f42a39347a4

// ---------- core file objects ----------

type object struct {
	typ  uint64
	body []byte
}

func parseObjects(d []byte) ([]object, error) {
	var out []object
	off := 0
	for off < len(d) {
		if off+12 > len(d) {
			return nil, fmt.Errorf("truncated object header")
		}
		t := binary.LittleEndian.Uint64(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+8:]))
		if off+12+sz > len(d) {
			return nil, fmt.Errorf("truncated object body")
		}
		b := make([]byte, sz)
		copy(b, d[off+12:off+12+sz])
		out = append(out, object{t, b})
		off += 12 + sz
	}
	return out, nil
}

func buildObjects(objs []object) []byte {
	var buf bytes.Buffer
	for _, o := range objs {
		var h [12]byte
		binary.LittleEndian.PutUint64(h[:], o.typ)
		binary.LittleEndian.PutUint32(h[8:], uint32(len(o.body)))
		buf.Write(h[:])
		buf.Write(o.body)
	}
	return buf.Bytes()
}

// ---------- SpawnSetup fields ----------

type ref struct {
	start, end int
	kind       byte
	uuid       []byte // 16 bytes, as stored
	path       string
}

type reader struct {
	b []byte
	o int
}

func (r *reader) u32() uint32 { v := binary.LittleEndian.Uint32(r.b[r.o:]); r.o += 4; return v }
func (r *reader) str() string {
	n := int(r.u32())
	if n == 0 {
		return ""
	}
	r.o += 4
	s := string(r.b[r.o : r.o+n])
	r.o += n
	return s
}
func (r *reader) ref() ref {
	s := r.o
	k := r.b[r.o]
	r.o++
	if k == 0 {
		return ref{s, r.o, 0, nil, ""}
	}
	u := r.b[r.o : r.o+16]
	r.o += 16
	p := ""
	if k == 2 || k == 3 || k == 5 {
		p = r.str()
	}
	return ref{s, r.o, k, u, p}
}

type ssFields struct {
	name                                     string
	nameEnd                                  int
	condition, impostor, entity, graph, fact ref
}

func spawnSetupFields(b []byte) ssFields {
	r := &reader{b, 16}
	f := ssFields{}
	f.name = r.str()
	f.nameEnd = r.o
	f.condition = r.ref()
	f.impostor = r.ref()
	f.entity = r.ref()
	f.graph = r.ref()
	f.fact = r.ref()
	return f
}

// ---------- machines ----------

type machine struct {
	code      string
	objs      []object
	ss        map[string]int
	normal    string
	corrupted string
}

// readData: game files the randomizer edits, by their old short names (read from the game, see gamedata.go)
func readData(name string) []byte {
	if name == "toughness.json" {
		return toughnessJSON()
	}
	b, err := gameFile(gamePathFor(name))
	if err != nil {
		panic(err)
	}
	return b
}

func gamePathFor(name string) string {
	switch {
	case strings.HasPrefix(name, "orig/herd_"):
		return "entities/herds/" + strings.TrimPrefix(name, "orig/herd_")
	case strings.HasPrefix(name, "orig/"):
		m := strings.TrimSuffix(strings.TrimPrefix(name, "orig/"), ".core")
		return "entities/spawnsetups/robots/" + m + "/" + m + ".core"
	case strings.HasPrefix(name, "dlc/"):
		m := strings.TrimSuffix(strings.TrimPrefix(name, "dlc/"), ".core")
		return "entities/dlc1/spawnsetups/robots/" + m + "/" + m + ".core"
	case strings.HasPrefix(name, "humans/"):
		k := strings.TrimSuffix(strings.TrimPrefix(name, "humans/"), ".core")
		return mapData.HTypes[k].Path + ".core"
	}
	return name
}

func loadMachine(code string) *machine {
	objs, err := parseObjects(readData("orig/" + code + ".core"))
	if err != nil {
		panic(code + ": " + err.Error())
	}
	m := &machine{code: code, objs: objs, ss: map[string]int{}}
	for i, o := range objs {
		if o.typ == spawnSetupType {
			m.ss[spawnSetupFields(o.body).name] = i
		}
	}
	for n := range m.ss {
		l := strings.ToLower(n)
		if l == code || l == code+"_spawnsetup" {
			m.normal = n
		}
		if l == code+"_corrupted" {
			m.corrupted = n
		}
	}
	return m
}

// uuid5 (RFC 4122, SHA-1) returned in little-endian GUID byte layout, matching the game files
func uuid5LE(name string) []byte {
	ns := []byte{0x6b, 0xa7, 0xb8, 0x12, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8} // NAMESPACE_OID
	h := sha1.New()
	h.Write(ns)
	h.Write([]byte(name))
	s := h.Sum(nil)[:16]
	s[6] = (s[6] & 0x0f) | 0x50
	s[8] = (s[8] & 0x3f) | 0x80
	le := make([]byte, 16)
	le[0], le[1], le[2], le[3] = s[3], s[2], s[1], s[0]
	le[4], le[5] = s[5], s[4]
	le[6], le[7] = s[7], s[6]
	copy(le[8:], s[8:])
	return le
}

func internalClosure(objs []object, start []byte) []int {
	var need []int
	seen := map[int]bool{}
	todo := [][]byte{start}
	for len(todo) > 0 {
		body := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		for i, o := range objs {
			if seen[i] {
				continue
			}
			pat := append([]byte{1}, o.body[:16]...)
			if bytes.Contains(body[16:], pat) {
				seen[i] = true
				need = append(need, i)
				todo = append(todo, o.body)
			}
		}
	}
	return need
}

func swapSlot(slotObjs []object, slotIdx int, donor *machine, donorName, tag string) []object {
	sb := slotObjs[slotIdx].body
	sf := spawnSetupFields(sb)
	db := donor.objs[donor.ss[donorName]].body
	df := spawnSetupFields(db)
	deps := internalClosure(donor.objs, db)
	type pair struct{ old, new []byte }
	var remap []pair
	for _, i := range deps {
		old := donor.objs[i].body[:16]
		remap = append(remap, pair{append([]byte{}, old...), uuid5LE(tag + fmt.Sprintf("%x", old))})
	}
	fix := func(b []byte) []byte {
		out := append([]byte{}, b...)
		for _, p := range remap {
			out = bytes.ReplaceAll(out, p.old, p.new)
		}
		return out
	}
	var nb []byte
	nb = append(nb, sb[:sf.nameEnd]...)               // slot UUID + slot name
	nb = append(nb, db[df.nameEnd:df.fact.start]...)  // donor fields up to faction
	nb = append(nb, sb[sf.fact.start:sf.fact.end]...) // slot faction
	nb = append(nb, db[df.fact.end:]...)              // donor rest
	nb = fix(nb)
	copy(nb[:16], sb[:16])
	slotObjs[slotIdx].body = nb
	var extra []object
	for _, i := range deps {
		extra = append(extra, object{donor.objs[i].typ, fix(donor.objs[i].body)})
	}
	return extra
}

// ---------- settings / mapping ----------

type Settings struct {
	Seed                      int64               `json:"seed"`
	Mode                      string              `json:"mode"`
	ChaoticKeepFlyersSeparate bool                `json:"chaotic_keep_flyers_separate"`
	GroupOrder                []string            `json:"group_order"`
	Groups                    map[string][]string `json:"groups"`
	Excluded                  []string            `json:"excluded"`
	GamePath                  string              `json:"game_path"`
}

func makeMapping(s Settings) map[string]string {
	rng := rand.New(rand.NewSource(s.Seed))
	ex := map[string]bool{}
	for _, e := range s.Excluded {
		ex[e] = true
	}
	filter := func(l []string) []string {
		var o []string
		for _, m := range l {
			if !ex[m] {
				o = append(o, m)
			}
		}
		return o
	}
	order := s.GroupOrder
	if len(order) == 0 {
		for k := range s.Groups {
			order = append(order, k)
		}
		sort.Strings(order)
	}
	var pools [][]string
	if s.Mode == "chaotic" {
		var all, fl []string
		flyer := map[string]bool{}
		if s.ChaoticKeepFlyersSeparate {
			for _, m := range filter(s.Groups["Flyer"]) {
				flyer[m] = true
				fl = append(fl, m)
			}
		}
		for _, g := range order {
			for _, m := range filter(s.Groups[g]) {
				if !flyer[m] {
					all = append(all, m)
				}
			}
		}
		pools = [][]string{all, fl}
	} else {
		for _, g := range order {
			pools = append(pools, filter(s.Groups[g]))
		}
	}
	mp := map[string]string{}
	for _, pool := range pools {
		if len(pool) < 2 {
			for _, m := range pool {
				mp[m] = m
			}
			continue
		}
		p := append([]string{}, pool...)
		for try := 0; try < 1000; try++ {
			rng.Shuffle(len(p), func(i, j int) { p[i], p[j] = p[j], p[i] })
			ok := true
			for i := range pool {
				if pool[i] == p[i] {
					ok = false
					break
				}
			}
			if ok {
				break
			}
		}
		for i := range pool {
			mp[pool[i]] = p[i]
		}
	}
	return mp
}

// ---------- prefetch ----------

type prefetch struct {
	hdrType  uint64
	uuid     []byte
	rawPaths [][]byte
	paths    []string
	sizes    []int32
	links    [][]int32
	index    map[string]int
}

func parsePrefetch(d []byte) *prefetch {
	p := &prefetch{index: map[string]int{}}
	p.hdrType = binary.LittleEndian.Uint64(d)
	b := d[12:]
	p.uuid = b[:16]
	off := 16
	n := int(binary.LittleEndian.Uint32(b[off:]))
	off += 4
	for i := 0; i < n; i++ {
		l := int(binary.LittleEndian.Uint32(b[off:]))
		p.rawPaths = append(p.rawPaths, b[off:off+8+l])
		s := string(b[off+8 : off+8+l])
		p.paths = append(p.paths, s)
		p.index[s] = i
		off += 8 + l
	}
	m := int(binary.LittleEndian.Uint32(b[off:]))
	off += 4
	p.sizes = make([]int32, m)
	for i := range p.sizes {
		p.sizes[i] = int32(binary.LittleEndian.Uint32(b[off:]))
		off += 4
	}
	k := int(binary.LittleEndian.Uint32(b[off:]))
	off += 4
	flat := make([]int32, k)
	for i := range flat {
		flat[i] = int32(binary.LittleEndian.Uint32(b[off:]))
		off += 4
	}
	for i := 0; i < len(flat); {
		c := int(flat[i])
		p.links = append(p.links, append([]int32{}, flat[i+1:i+1+c]...))
		i += 1 + c
	}
	return p
}

func (p *prefetch) serialize() []byte {
	var body bytes.Buffer
	w32 := func(v uint32) { var t [4]byte; binary.LittleEndian.PutUint32(t[:], v); body.Write(t[:]) }
	body.Write(p.uuid)
	w32(uint32(len(p.paths)))
	for _, r := range p.rawPaths {
		body.Write(r)
	}
	w32(uint32(len(p.sizes)))
	for _, s := range p.sizes {
		w32(uint32(s))
	}
	var flat []int32
	for _, l := range p.links {
		flat = append(flat, int32(len(l)))
		flat = append(flat, l...)
	}
	w32(uint32(len(flat)))
	for _, v := range flat {
		w32(uint32(v))
	}
	var out bytes.Buffer
	var h [12]byte
	binary.LittleEndian.PutUint64(h[:], p.hdrType)
	binary.LittleEndian.PutUint32(h[8:], uint32(body.Len()))
	out.Write(h[:])
	out.Write(body.Bytes())
	return out.Bytes()
}

func loadPrefetch() *prefetch {
	d, err := gameFile("prefetch/fullgame.prefetch.core")
	if err != nil {
		panic(err)
	}
	return parsePrefetch(d)
}

// external links (kind 2) referenced in a core file
func externalPaths(b []byte) []string {
	var out []string
	for i := 0; i+25 < len(b); i++ {
		if b[i] != 2 {
			continue
		}
		l := int(binary.LittleEndian.Uint32(b[i+17:]))
		if l <= 0 || l > 300 || i+25+l > len(b) {
			continue
		}
		s := string(b[i+25 : i+25+l])
		for _, pre := range []string{"entities/", "ai/", "interface/", "models/", "sounds/", "levels/"} {
			if strings.HasPrefix(s, pre) {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

// ---------- generator ----------

var allMachines = []string{"antelope", "beachlizard", "bison", "cargorhino", "crab", "direwolf", "glider", "goat", "greywolf", "harvester", "horse", "hyena", "laserscout", "longhorn", "longleg", "raptor", "scout", "spraybot", "stalker", "thunderhawk", "mole", "hackbot", "warrobot", "comgiraffe"}

type outFile struct {
	path string
	data []byte
}

func generate(s Settings, mp map[string]string) ([]outFile, error) {
	return generateParts(s, mp, true, true, nil)
}

func generateParts(s Settings, mp map[string]string, doSpawn, doHerds bool, extra []outFile) ([]outFile, error) {
	M := map[string]*machine{}
	for m := range mp {
		M[m] = loadMachine(m)
	}
	// raw path strings (with stored CRC) for every machine entity path
	strBytes := map[string][]byte{}
	for _, m := range allMachines {
		d := readData("orig/" + m + ".core")
		for i := 0; i+8 < len(d); i++ {
			l := int(binary.LittleEndian.Uint32(d[i:]))
			if l < 20 || l > 120 || i+8+l > len(d) {
				continue
			}
			p := string(d[i+8 : i+8+l])
			if strings.HasPrefix(p, "entities/characters/robots/") && !strings.ContainsAny(p, "\x00:") {
				strBytes[p] = append([]byte{}, d[i:i+8+l]...)
			}
		}
	}
	out := append([]outFile{}, extra...)
	keys := make([]string, 0, len(mp))
	for k := range mp {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, slot := range keys {
		donor := mp[slot]
		if slot == donor || !doSpawn {
			continue
		}
		sm := M[slot]
		objs := make([]object, len(sm.objs))
		for i, o := range sm.objs {
			objs[i] = object{o.typ, append([]byte{}, o.body...)}
		}
		var extra []object
		for _, kind := range []string{"normal", "corrupted"} {
			sn, dn := sm.normal, M[donor].normal
			if kind == "corrupted" {
				sn = sm.corrupted
				if M[donor].corrupted != "" {
					dn = M[donor].corrupted
				}
			}
			if sn != "" && dn != "" {
				extra = append(extra, swapSlot(objs, sm.ss[sn], M[donor], dn, fmt.Sprintf("%d:%s:%s:", s.Seed, slot, kind))...)
			}
		}
		out = append(out, outFile{"entities/spawnsetups/robots/" + slot + "/" + slot + ".core", buildObjects(append(objs, extra...))})
	}
	// herd lists
	type ent struct {
		uuid []byte
		path string
	}
	entOf := func(m, kind string) (ent, bool) {
		n := M[m].normal
		if kind == "corrupted" {
			n = M[m].corrupted
		}
		if n == "" {
			return ent{}, false
		}
		f := spawnSetupFields(M[m].objs[M[m].ss[n]].body)
		return ent{f.entity.uuid, f.entity.path}, true
	}
	swap := map[string]ent{}
	for m, d := range mp {
		for _, kind := range []string{"normal", "corrupted"} {
			a, ok1 := entOf(m, kind)
			b, ok2 := entOf(d, kind)
			if ok1 && ok2 {
				swap[string(a.uuid)+a.path] = b
			}
		}
	}
	herdList := []string{"resourcegatheringherd", "recongroup"}
	if !doHerds {
		herdList = nil
	}
	for _, h := range herdList {
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
						p := string(d[i+25 : i+25+l])
						if n, ok := swap[string(d[i+1:i+17])+p]; ok {
							sb, ok2 := strBytes[n.path]
							if !ok2 {
								return nil, fmt.Errorf("missing path bytes for %s", n.path)
							}
							res = append(res, 3)
							res = append(res, n.uuid...)
							res = append(res, sb...)
							i += 25 + l
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
	// prefetch
	pf := loadPrefetch()
	for _, f := range out {
		key := strings.TrimSuffix(f.path, ".core")
		i, ok := pf.index[key]
		if !ok {
			return nil, fmt.Errorf("prefetch missing %s", key)
		}
		pf.sizes[i] = int32(len(f.data))
		if strings.HasPrefix(f.path, "entities/spawnsetups") || strings.HasPrefix(f.path, "entities/dlc1/spawnsetups") || strings.HasPrefix(f.path, "levels/") {
			set := map[int32]bool{}
			for _, l := range pf.links[i] {
				set[l] = true
			}
			for _, r := range externalPaths(f.data) {
				if j, ok := pf.index[r]; ok {
					set[int32(j)] = true
				}
			}
			var ls []int32
			for l := range set {
				ls = append(ls, l)
			}
			sort.Slice(ls, func(a, b int) bool { return ls[a] < ls[b] })
			pf.links[i] = ls
		}
	}
	out = append(out, outFile{"prefetch/fullgame.prefetch.core", pf.serialize()})
	// sanity: every output core must re-parse
	for _, f := range out {
		if _, err := parseObjects(f.data); err != nil {
			return nil, fmt.Errorf("internal check failed for %s: %v", f.path, err)
		}
	}
	return out, nil
}

// ---------- archive writer (uncompressed Kraken blocks: readable by the game's old Oodle) ----------

func packRaw(files []outFile) []byte {
	const ch = 0x40000
	var data bytes.Buffer
	type fe struct {
		idx       uint32
		hash, off uint64
		size      uint32
	}
	var ents []fe
	for i, f := range files {
		ents = append(ents, fe{uint32(i), pathHash(f.path), uint64(data.Len()), uint32(len(f.data))})
		data.Write(f.data)
	}
	sort.Slice(ents, func(a, b int) bool { return ents[a].hash < ents[b].hash })
	raw := data.Bytes()
	nch := (len(raw) + ch - 1) / ch
	co := uint64(40 + 32*len(ents) + 32*nch)
	var tab, blobs bytes.Buffer
	le := binary.LittleEndian
	for c := 0; c < nch; c++ {
		end := (c + 1) * ch
		if end > len(raw) {
			end = len(raw)
		}
		blk := raw[c*ch : end]
		var e [32]byte
		le.PutUint64(e[0:], uint64(c*ch))
		le.PutUint32(e[8:], uint32(len(blk)))
		le.PutUint64(e[16:], co)
		le.PutUint32(e[24:], uint32(len(blk)+2))
		tab.Write(e[:])
		blobs.Write([]byte{0xcc, 0x06})
		blobs.Write(blk)
		co += uint64(len(blk) + 2)
	}
	var out bytes.Buffer
	var h [40]byte
	le.PutUint32(h[0:], 0x20304050)
	le.PutUint64(h[8:], co)
	le.PutUint64(h[16:], uint64(len(raw)))
	le.PutUint64(h[24:], uint64(len(ents)))
	le.PutUint32(h[32:], uint32(nch))
	le.PutUint32(h[36:], ch)
	out.Write(h[:])
	for _, e := range ents {
		var b [32]byte
		le.PutUint32(b[0:], e.idx)
		le.PutUint64(b[8:], e.hash)
		le.PutUint64(b[16:], e.off)
		le.PutUint32(b[24:], e.size)
		out.Write(b[:])
	}
	out.Write(tab.Bytes())
	out.Write(blobs.Bytes())
	return out.Bytes()
}
