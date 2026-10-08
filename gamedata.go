package main

// Game data comes from the player's own game folder - the program carries none of it.
// On first start (or when the game folder changes) the needed files are read out of the game's
// archives, checked against the fingerprints in mapdata/manifest.json, and kept in a local cache
// file next to the program so later starts are quick.

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type manifestFile struct {
	Path string `json:"path"`
	Size int    `json:"size"`
	Sha  string `json:"sha"`
}

type manifestTile struct {
	X    int    `json:"x"`
	Y    int    `json:"y"`
	DLC  bool   `json:"dlc"`
	Path string `json:"path"`
}

type gameManifest struct {
	Files []manifestFile    `json:"files"`
	Tiles []manifestTile    `json:"tiles"`
	Icons map[string]string `json:"icons"`
}

var errNoGameFolder = fmt.Errorf("the game folder isn't set")

const gameCacheName = "EnemyRandomizer_gamedata.cache"
const gameCacheFormat = "hzd-randomizer-cache-1"

var gameData struct {
	sync.Mutex
	files       map[string][]byte // game path -> file
	images      map[string][]byte // "worldmap.jpg", "<icon>.png"
	ready       bool
	loading     bool
	err         string
	stage       string
	done, total int
}

func loadManifest() gameManifest {
	var m gameManifest
	b, err := mapFS.ReadFile("mapdata/manifest.json")
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, &m); err != nil {
		panic(err)
	}
	return m
}

func manifestID() string {
	b, _ := mapFS.ReadFile("mapdata/manifest.json")
	h := sha256.Sum256(append([]byte(gameCacheFormat), b...))
	return hex.EncodeToString(h[:8])
}

func cachePath() string {
	exe, err := os.Executable()
	if err != nil {
		return gameCacheName
	}
	return filepath.Join(filepath.Dir(exe), gameCacheName)
}

func setStage(stage string, done, total int) {
	gameData.Lock()
	gameData.stage, gameData.done, gameData.total = stage, done, total
	gameData.Unlock()
}

// gameStatus is shown on the page while the game files load
func gameStatus() map[string]interface{} {
	gameData.Lock()
	defer gameData.Unlock()
	return map[string]interface{}{"ready": gameData.ready, "loading": gameData.loading, "error": gameData.err,
		"stage": gameData.stage, "done": gameData.done, "total": gameData.total}
}

func gameReady() bool {
	gameData.Lock()
	defer gameData.Unlock()
	return gameData.ready
}

// startGameDataLoad loads in the background (cache first, then the game's archives)
func startGameDataLoad(packedDir string, fresh bool) {
	gameData.Lock()
	if gameData.loading {
		gameData.Unlock()
		return
	}
	gameData.loading, gameData.err = true, ""
	gameData.Unlock()
	go func() {
		err := loadGameData(packedDir, fresh)
		gameData.Lock()
		gameData.loading = false
		if err != nil && err != errNoGameFolder {
			gameData.err = err.Error()
		}
		gameData.Unlock()
	}()
}

func loadGameData(packedDir string, fresh bool) error {
	if !fresh {
		if files, imgs, ok := readGameCache(cachePath()); ok {
			publishGameData(files, imgs)
			return nil
		}
	}
	if !isPacked(packedDir) {
		return errNoGameFolder
	}
	files, imgs, err := extractGameData(packedDir)
	if err != nil {
		return err
	}
	publishGameData(files, imgs)
	if err := writeGameCache(cachePath(), files, imgs); err != nil {
		fmt.Println("Note: couldn't save the game-data cache:", err)
	}
	return nil
}

func publishGameData(files, imgs map[string][]byte) {
	gameData.Lock()
	gameData.files, gameData.images, gameData.ready, gameData.err = files, imgs, true, ""
	gameData.stage = "ready"
	gameData.Unlock()
	sceneCache = map[string][]byte{}
}

func extractGameData(packedDir string) (map[string][]byte, map[string][]byte, error) {
	man := loadManifest()
	setStage("Opening the game's archives", 0, 1)
	if err := initDecompressor(packedDir); err != nil {
		return nil, nil, err
	}
	arc, err := openArchiveSet(packedDir)
	if err != nil {
		return nil, nil, err
	}
	defer arc.close()
	extra := os.Getenv("HZD_EXTRA_FILES") // developer testing only
	read := func(p string) ([]byte, error) {
		b, err := arc.read(p)
		if err != nil && extra != "" {
			if b2, err2 := os.ReadFile(filepath.Join(extra, filepath.FromSlash(p))); err2 == nil {
				return b2, nil
			}
		}
		return b, err
	}
	total := len(man.Files) + len(man.Tiles) + len(man.Icons)
	files := map[string][]byte{}
	slow, _ := strconv.Atoi(os.Getenv("HZD_DEV_DELAY_MS")) // developer testing only
	for i, f := range man.Files {
		setStage("Reading game files", i, total)
		if slow > 0 {
			time.Sleep(time.Duration(slow) * time.Millisecond)
		}
		b, err := read(f.Path)
		if err != nil {
			return nil, nil, err
		}
		h := sha256.Sum256(b)
		if hex.EncodeToString(h[:]) != f.Sha {
			return nil, nil, fmt.Errorf("%s is different from the game version this tool was made for (Horizon Zero Dawn Complete Edition, original PC release, latest patch). If the game was updated, verify the game files in Steam/GOG", f.Path)
		}
		files[f.Path] = b
	}
	imgs := map[string][]byte{}
	done := len(man.Files)
	// world map: one 256x256 picture per map tile
	M := mapData.Map
	tp, xmin, xmax, ymin, ymax := M["tile_px"], M["xmin"], M["xmax"], M["ymin"], M["ymax"]
	W, H := (xmax-xmin+1)*tp, (ymax-ymin+1)*tp
	canvas := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.RGBA{14, 19, 22, 255}}, image.Point{}, draw.Src)
	placed := map[[2]int]bool{}
	tiles := append([]manifestTile{}, man.Tiles...)
	// Frozen Wilds tiles win where both games have one
	for pass := 0; pass < 2; pass++ {
		for _, t := range tiles {
			if (pass == 0) != t.DLC || placed[[2]int{t.X, t.Y}] {
				continue
			}
			done++
			setStage("Drawing the world map", done, total)
			img, err := mapTile(read, t.Path, tp)
			if err != nil {
				return nil, nil, err
			}
			at := image.Pt((t.X-xmin)*tp, (ymax-t.Y)*tp)
			draw.Draw(canvas, image.Rectangle{at, at.Add(img.Bounds().Size())}, img, image.Point{}, draw.Over)
			placed[[2]int{t.X, t.Y}] = true
		}
	}
	var jb bytes.Buffer
	if err := jpeg.Encode(&jb, canvas, &jpeg.Options{Quality: 88}); err != nil {
		return nil, nil, err
	}
	imgs["worldmap.jpg"] = jb.Bytes()
	// map-marker icons
	for code, p := range man.Icons {
		done++
		setStage("Reading map icons", done, total)
		b, err := read(p)
		if err != nil {
			return nil, nil, err
		}
		img, err := markerIcon(b)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %v", p, err)
		}
		imgs[code+".png"] = encodePNG(img)
		if code == "scout" {
			imgs["laserscout.png"] = imgs[code+".png"] // the Redeye Watcher shares the Watcher icon
		}
		if code == "cauldron_5" {
			imgs["cauldron_epsilon.png"] = imgs[code+".png"]
		}
		if code == "spraybot" { // Freeze Bellowback: the Bellowback icon tinted icy blue
			imgs["spraybot_cryo.png"] = encodePNG(tint(img, 0.68, 0.97, 1.08))
		}
	}
	setStage("ready", total, total)
	return files, imgs, nil
}

func encodePNG(img image.Image) []byte {
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

func tint(src *image.NRGBA, r, g, b float64) *image.NRGBA {
	out := image.NewNRGBA(src.Bounds())
	clamp := func(v float64) uint8 {
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	for i := 0; i+3 < len(src.Pix); i += 4 {
		out.Pix[i] = clamp(float64(src.Pix[i]) * r)
		out.Pix[i+1] = clamp(float64(src.Pix[i+1]) * g)
		out.Pix[i+2] = clamp(float64(src.Pix[i+2]) * b)
		out.Pix[i+3] = src.Pix[i+3]
	}
	return out
}

// mapTile decodes the tile's in-game map texture at the mip level matching the map's tile size
func mapTile(read func(string) ([]byte, error), path string, size int) (image.Image, error) {
	b, err := read(path)
	if err != nil {
		return nil, err
	}
	objs, err := parseObjects(b)
	if err != nil || len(objs) == 0 {
		return nil, fmt.Errorf("%s: not a texture", path)
	}
	body := objs[0].body
	t, _, err := readTexture(body, skipString(body, 16))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	if t.format != pfBC1 || t.w != t.h {
		return nil, fmt.Errorf("%s: unexpected map texture", path)
	}
	// mips are stored largest first; the stream file holds the first extMips of them
	off, w := 0, t.w
	for w > size {
		off += w * w / 2
		w /= 2
	}
	var data []byte
	if t.extMips > 0 && off < t.extSize {
		s, err := read(path + ".stream")
		if err != nil {
			return nil, err
		}
		if off+w*w/2 > len(s) {
			return nil, fmt.Errorf("%s: map texture stream too short", path)
		}
		data = s[off : off+w*w/2]
	} else {
		o := off - t.extSize
		if o < 0 || o+w*w/2 > len(t.internal) {
			return nil, fmt.Errorf("%s: map texture data missing", path)
		}
		data = t.internal[o : o+w*w/2]
	}
	return decodeTexture(w, w, pfBC1, data)
}

// markerIcon decodes the small (64x64) picture of a UITexture map marker
func markerIcon(b []byte) (*image.NRGBA, error) {
	objs, err := parseObjects(b)
	if err != nil || len(objs) == 0 {
		return nil, fmt.Errorf("not a texture")
	}
	body := objs[0].body
	o := skipString(body, 16) // Name
	o = skipString(body, o)   // TextureName
	o += 8                    // Size
	if o+8 > len(body) {
		return nil, fmt.Errorf("icon cut short")
	}
	small := binary.LittleEndian.Uint32(body[o:])
	o += 8
	if small == 0 {
		return nil, fmt.Errorf("icon has no small picture")
	}
	t, _, err := readTexture(body, o)
	if err != nil {
		return nil, err
	}
	return decodeTexture(t.w, t.h, t.format, t.internal)
}

// ---------- cache file (zip next to the program) ----------

func writeGameCache(path string, files, imgs map[string][]byte) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	put := func(name string, b []byte) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	}
	if err := put("cache-id", []byte(manifestID())); err != nil {
		f.Close()
		return err
	}
	for p, b := range files {
		if err := put("files/"+p, b); err != nil {
			f.Close()
			return err
		}
	}
	for n, b := range imgs {
		if err := put("img/"+n, b); err != nil {
			f.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		f.Close()
		return err
	}
	f.Close()
	os.Remove(path)
	return os.Rename(tmp, path)
}

func readGameCache(path string) (map[string][]byte, map[string][]byte, bool) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, nil, false
	}
	defer zr.Close()
	files, imgs := map[string][]byte{}, map[string][]byte{}
	id := ""
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, nil, false
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, nil, false
		}
		switch {
		case f.Name == "cache-id":
			id = string(b)
		case strings.HasPrefix(f.Name, "files/"):
			files[strings.TrimPrefix(f.Name, "files/")] = b
		case strings.HasPrefix(f.Name, "img/"):
			imgs[strings.TrimPrefix(f.Name, "img/")] = b
		}
	}
	if id != manifestID() {
		return nil, nil, false
	}
	// quick check that nothing is missing or changed
	for _, m := range loadManifest().Files {
		b, ok := files[m.Path]
		if !ok || len(b) != m.Size {
			return nil, nil, false
		}
	}
	if _, ok := imgs["worldmap.jpg"]; !ok {
		return nil, nil, false
	}
	return files, imgs, true
}

// ---------- access used by the randomizer ----------

func gameFile(path string) ([]byte, error) {
	gameData.Lock()
	defer gameData.Unlock()
	if !gameData.ready {
		return nil, fmt.Errorf("the game files haven't been read yet")
	}
	b, ok := gameData.files[path]
	if !ok {
		return nil, fmt.Errorf("missing game file %s", path)
	}
	return b, nil
}

func gameImage(name string) ([]byte, bool) {
	gameData.Lock()
	defer gameData.Unlock()
	b, ok := gameData.images[name]
	return b, ok
}

// toughnessJSON: the health/damage modifier templates are copied from two bandit spawn files in
// the game (by their object id); the rest of toughness_src.json is just ids and names.
func toughnessJSON() []byte {
	var t map[string]interface{}
	b, err := dataFS.ReadFile("data/toughness_src.json")
	if err != nil {
		panic(err)
	}
	json.Unmarshal(b, &t)
	for _, k := range []string{"dm", "dd"} {
		src := t[k+"_from"].(map[string]interface{})
		f, err := gameFile(src["path"].(string))
		if err != nil {
			panic(err)
		}
		objs, err := parseObjects(f)
		if err != nil {
			panic(err)
		}
		guid, _ := hex.DecodeString(src["guid"].(string))
		found := false
		for _, o := range objs {
			if len(o.body) == int(src["size"].(float64)) && bytes.Equal(o.body[:16], guid) {
				t[k] = hex.EncodeToString(o.body)
				found = true
				break
			}
		}
		if !found {
			panic("toughness template not found in " + src["path"].(string))
		}
	}
	out, _ := json.Marshal(t)
	return out
}
