package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const version = "1.1.0"
const mapPatchName = "Patch_EnemyRandomizer_Map.bin"
const configName = "EnemyRandomizer_Map_settings.json"

//go:embed web
var webFS embed.FS

var names = map[string]string{
	"harvester": "Grazer", "scout": "Watcher", "laserscout": "Redeye Watcher", "horse": "Strider", "longhorn": "Broadhead",
	"antelope": "Lancehorn", "goat": "Charger", "hyena": "Scrapper", "glider": "Glinthawk", "beachlizard": "Snapmaw",
	"direwolf": "Sawtooth", "greywolf": "Ravager", "bison": "Trampler", "longleg": "Longleg", "stalker": "Stalker",
	"spraybot": "Fire Bellowback", "crab": "Shell-Walker", "mole": "Rockbreaker", "cargorhino": "Behemoth",
	"thunderhawk": "Stormbird", "raptor": "Thunderjaw", "hackbot": "Corruptor", "warrobot": "Deathbringer", "comgiraffe": "Tallneck",
	"hellhound": "Scorcher", "cryobear": "Frostclaw", "infernobear": "Fireclaw", "spraybot_cryo": "Freeze Bellowback",
}

func defaultSettings() Settings {
	return Settings{
		Mode: "grouped", ChaoticKeepFlyersSeparate: true,
		GroupOrder: []string{"Small", "Medium", "Large", "Flyer"},
		Groups: map[string][]string{
			"Small":  {"harvester", "scout", "laserscout", "horse", "antelope", "hyena"},
			"Medium": {"longhorn", "goat", "direwolf", "greywolf", "beachlizard", "bison", "longleg", "stalker", "spraybot", "spraybot_cryo", "crab", "hellhound"},
			"Large":  {"cargorhino", "raptor", "cryobear", "infernobear"},
			"Flyer":  {"glider", "thunderhawk"},
		},
		Excluded: []string{"comgiraffe", "mole", "hackbot", "warrobot"},
	}
}

func exeDir() string {
	p, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(p)
}

func isPacked(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "Initial.bin"))
	return err == nil
}

func findGame(saved string) string {
	var cands []string
	if saved != "" {
		cands = append(cands, saved)
	}
	ed := exeDir()
	cands = append(cands, ed, filepath.Join(ed, "Packed_DX12"), filepath.Join(ed, "..", "Packed_DX12"))
	roots := []string{`C:\Program Files (x86)\Steam`, `C:\Program Files\Steam`}
	for _, d := range "CDEFGHIJKLMNOPQRSTUVWXYZ" {
		roots = append(roots, fmt.Sprintf(`%c:\SteamLibrary`, d), fmt.Sprintf(`%c:\Steam`, d), fmt.Sprintf(`%c:\Games\Steam`, d), fmt.Sprintf(`%c:\Games\SteamLibrary`, d))
	}
	re := regexp.MustCompile(`"path"\s+"([^"]+)"`)
	for _, r := range []string{`C:\Program Files (x86)\Steam`, `C:\Program Files\Steam`} {
		if b, err := os.ReadFile(filepath.Join(r, "steamapps", "libraryfolders.vdf")); err == nil {
			for _, m := range re.FindAllStringSubmatch(string(b), -1) {
				roots = append(roots, strings.ReplaceAll(m[1], `\\`, `\`))
			}
		}
	}
	for _, r := range roots {
		cands = append(cands, filepath.Join(r, "steamapps", "common", "Horizon Zero Dawn", "Packed_DX12"))
	}
	cands = append(cands, `C:\Program Files\Epic Games\HorizonZeroDawn\Packed_DX12`)
	for _, c := range cands {
		if isPacked(c) {
			return c
		}
		if isPacked(filepath.Join(c, "Packed_DX12")) {
			return filepath.Join(c, "Packed_DX12")
		}
	}
	return ""
}

var cfgMu sync.Mutex

func loadConfig() MapConfig {
	c := defaultMapConfig()
	if b, err := os.ReadFile(filepath.Join(exeDir(), configName)); err == nil {
		if json.Unmarshal(b, &c) != nil {
			c = defaultMapConfig()
		}
	}
	c.migrate()
	return c
}

func saveConfig(c MapConfig) error {
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(filepath.Join(exeDir(), configName), b, 0644)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func main() {
	loadMapData()
	if len(os.Args) == 4 && os.Args[1] == "-build" { // developer: -build config.json out.bin (game folder in HZD_PACKED, or the cache)
		if err := loadGameData(os.Getenv("HZD_PACKED"), false); err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}
		var c MapConfig
		b, _ := os.ReadFile(os.Args[2])
		if err := json.Unmarshal(b, &c); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		c.migrate()
		data, ch, err := buildMapPatch(c)
		if err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}
		os.WriteFile(os.Args[3], data, 0644)
		fmt.Println("changes:", len(ch))
		return
	}
	cfg := loadConfig()
	// a saved folder from another PC that isn't here: look for the game, but keep the old path on screen if nothing is found
	if g := findGame(cfg.GamePath); g != "" {
		cfg.GamePath = g
	}
	saveConfig(cfg)
	startGameDataLoad(cfg.GamePath, false) // the cache, or the game's own files

	mux := http.NewServeMux()
	gameInfo := func() map[string]interface{} {
		valid := cfg.GamePath != "" && isPacked(cfg.GamePath)
		installed := false
		if valid {
			_, err := os.Stat(filepath.Join(cfg.GamePath, mapPatchName))
			installed = err == nil
		}
		return map[string]interface{}{"ok": true, "path": cfg.GamePath, "valid": valid, "installed": installed}
	}
	mux.HandleFunc("/img/", func(w http.ResponseWriter, r *http.Request) {
		// map picture and machine icons are made from the player's own game files (gamedata.go)
		b, ok := gameImage(strings.TrimPrefix(r.URL.Path, "/img/"))
		if !ok {
			var err error
			if b, err = webFS.ReadFile("web" + r.URL.Path); err != nil {
				http.NotFound(w, r)
				return
			}
		}
		if strings.HasSuffix(r.URL.Path, ".jpg") {
			w.Header().Set("Content-Type", "image/jpeg")
		} else {
			w.Header().Set("Content-Type", "image/png")
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(b)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, _ := webFS.ReadFile("web/index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(b)
	})
	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		cfgMu.Lock()
		defer cfgMu.Unlock()
		gi := gameInfo()
		installed := gi["installed"]
		writeJSON(w, map[string]interface{}{"game_valid": gi["valid"], "sites": mapData.Sites, "map": mapData.Map, "cauldrons": mapData.Cauldrons, "humans": mapData.Humans, "htypes": mapData.HTypes, "regions": mapData.Regions, "std": mapData.Std,
			"names": names, "config": cfg, "installed": installed, "version": version, "gamedata": gameStatus()})
	})
	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		var c MapConfig
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "message": err.Error()})
			return
		}
		c.migrate()
		cfgMu.Lock()
		c.GamePath = cfg.GamePath
		c.History = cfg.History // build history is kept by the program, not the page
		cfg = c
		saveConfig(cfg)
		cfgMu.Unlock()
		writeJSON(w, map[string]interface{}{"ok": true})
	})
	setGame := func(w http.ResponseWriter, raw string) {
		p, ok := resolveGameFolder(raw)
		if !ok {
			writeJSON(w, map[string]interface{}{"ok": false, "message": "That folder isn't the game folder - it should contain Initial.bin (pick Horizon Zero Dawn or its Packed_DX12 folder)."})
			return
		}
		cfgMu.Lock()
		cfg.GamePath = p
		saveConfig(cfg)
		r := gameInfo()
		cfgMu.Unlock()
		if !gameReady() {
			startGameDataLoad(p, false)
		}
		r["message"] = "Game folder set: " + p
		writeJSON(w, r)
	}
	mux.HandleFunc("/api/gamepath", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Path string }
		json.NewDecoder(r.Body).Decode(&req)
		setGame(w, req.Path)
	})
	mux.HandleFunc("/api/browse", func(w http.ResponseWriter, r *http.Request) {
		cfgMu.Lock()
		start := cfg.GamePath
		cfgMu.Unlock()
		p, err := browseFolder(start)
		if err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "message": "Couldn't open the folder picker - paste the path instead."})
			return
		}
		if p == "" {
			writeJSON(w, map[string]interface{}{"ok": false, "cancelled": true})
			return
		}
		setGame(w, p)
	})
	mux.HandleFunc("/api/detect", func(w http.ResponseWriter, r *http.Request) {
		p := findGame("")
		if p == "" {
			writeJSON(w, map[string]interface{}{"ok": false, "message": "Couldn't find the game automatically - use Browse or paste the path."})
			return
		}
		setGame(w, p)
	})
	mux.HandleFunc("/api/gamestatus", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, gameStatus())
	})
	mux.HandleFunc("/api/reload", func(w http.ResponseWriter, r *http.Request) { // re-read the game files (e.g. after a game update)
		cfgMu.Lock()
		p := cfg.GamePath
		cfgMu.Unlock()
		if !isPacked(p) {
			writeJSON(w, map[string]interface{}{"ok": false, "message": "Set the game folder first."})
			return
		}
		startGameDataLoad(p, true)
		writeJSON(w, map[string]interface{}{"ok": true})
	})
	mux.HandleFunc("/api/preview", func(w http.ResponseWriter, r *http.Request) {
		if !gameReady() {
			writeJSON(w, map[string]interface{}{"ok": false, "message": notReadyMsg()})
			return
		}
		var c MapConfig
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "message": err.Error()})
			return
		}
		c.migrate()
		writeJSON(w, previewMap(c))
	})
	mux.HandleFunc("/api/build", func(w http.ResponseWriter, r *http.Request) {
		var c MapConfig
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "message": err.Error()})
			return
		}
		if !gameReady() {
			writeJSON(w, map[string]interface{}{"ok": false, "message": notReadyMsg()})
			return
		}
		cfgMu.Lock()
		defer cfgMu.Unlock()
		c.migrate()
		c.GamePath = cfg.GamePath
		c.History = cfg.History
		cfg = c
		saveConfig(cfg)
		if cfg.GamePath == "" || !isPacked(cfg.GamePath) {
			writeJSON(w, map[string]interface{}{"ok": false, "message": "The game folder isn't set or can't be found on this PC - set it under Game folder first."})
			return
		}
		data, changes, err := buildMapPatch(cfg)
		if err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "message": "Build failed: " + err.Error()})
			return
		}
		notes := []string{}
		for _, old := range []string{"Patch_EnemyRandomizer.bin", "Patch_EnemyRandomizer_Grouped.bin", "Patch_EnemyRandomizer_Chaotic.bin"} {
			if os.Remove(filepath.Join(cfg.GamePath, old)) == nil {
				notes = append(notes, "Removed "+old+" (the whole-world randomizer) - only one randomizer can be active.")
			}
		}
		if _, err := os.Stat(filepath.Join(cfg.GamePath, "Patch_New.bin")); err == nil {
			notes = append(notes, "Patch_New.bin (old test patch) is in the game folder and will clash - rename it to Patch_New.bin.off.")
		}
		if err := os.WriteFile(filepath.Join(cfg.GamePath, mapPatchName), data, 0644); err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "message": "Could not write the patch (is the game running?): " + err.Error()})
			return
		}
		mode := "Areas"
		if cfg.Whole.On {
			mode = "Whole map " + cfg.wholeArea().Mode + fmt.Sprintf(", seed %d", cfg.Whole.Seed)
		}
		h := HistoryEntry{Time: time.Now().Format("Mon 2 Jan 15:04"), Changes: len(changes), Summary: mode, Config: cfg.snapshot()}
		cfg.History = append([]HistoryEntry{h}, cfg.History...)
		if len(cfg.History) > 10 {
			cfg.History = cfg.History[:10]
		}
		saveConfig(cfg)
		writeJSON(w, map[string]interface{}{"ok": true, "changes": changes, "notes": notes, "history": cfg.History,
			"message": fmt.Sprintf("Installed %s - %d sites changed.", mapPatchName, len(changes)) + toughNote(cfg)})
	})
	mux.HandleFunc("/api/uninstall", func(w http.ResponseWriter, r *http.Request) {
		cfgMu.Lock()
		defer cfgMu.Unlock()
		if cfg.GamePath != "" && os.Remove(filepath.Join(cfg.GamePath, mapPatchName)) == nil {
			writeJSON(w, map[string]interface{}{"ok": true, "message": "Removed. The game is back to normal."})
			return
		}
		writeJSON(w, map[string]interface{}{"ok": true, "message": "The map randomizer wasn't installed."})
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("Could not start:", err)
		fmt.Scanln()
		return
	}
	url := fmt.Sprintf("http://%s/", ln.Addr().String())
	fmt.Println("==============================================")
	fmt.Println("  Horizon Zero Dawn - Enemy Randomizer: Map v" + version)
	fmt.Println("==============================================")
	if isPacked(cfg.GamePath) {
		fmt.Println("Game folder:", cfg.GamePath)
	} else {
		fmt.Println("Game folder not found on this PC - set it in the browser page (Game folder > Browse).")
	}
	fmt.Println()
	fmt.Println("The map is opening in your web browser.")
	fmt.Println("If it doesn't, open this address:", url)
	fmt.Println()
	fmt.Println("Keep this window open while you use the map. Close it when you're done.")
	go func() { time.Sleep(400 * time.Millisecond); openBrowser(url) }()
	http.Serve(ln, mux)
}

func toughNote(c MapConfig) string {
	var p []string
	if c.Toughness.active() {
		p = append(p, "machine")
	}
	if c.HumanTough.active() {
		p = append(p, "human")
	}
	if len(p) == 0 {
		return ""
	}
	return " " + strings.ToUpper(p[0][:1]) + strings.Join(p, " and ")[1:] + " toughness changes applied."
}

func notReadyMsg() string {
	st := gameStatus()
	if e, _ := st["error"].(string); e != "" {
		return "Couldn't read the game files: " + e
	}
	if l, _ := st["loading"].(bool); l {
		return "Still reading your game files - try again in a moment."
	}
	return "Set the game folder first (Setup tab) so the randomizer can read your game files."
}
