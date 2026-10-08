Horizon Zero Dawn - Enemy Randomizer: Map v1.1.0
For Horizon Zero Dawn Complete Edition on PC (the original, not Remastered).

Changes which machines and human enemies you meet, where - picked on the game's own world map.
Free fan-made tool. Not affiliated with Guerrilla Games or Sony.


WHAT'S NEW IN 1.1.0
 - The program no longer contains any files from the game. The map, the machine icons and the
   game data it edits are now read from YOUR copy of Horizon Zero Dawn the first time you start
   it (or after you pick the game folder). This takes a moment once; after that they're kept in
   EnemyRandomizer_gamedata.cache next to the program, so later starts are instant.
 - That makes the program itself safe to share. (The cache file does contain game data - don't
   share that file.)
 - The program is much smaller (about 11 MB instead of 25 MB).
 - Every patch it builds is byte-for-byte the same as 1.0.0 with the same settings and seed.
 - The world map is drawn on a plain dark background (the game's decorative border art around
   the map is no longer included).
 - New: Setup tab > "Re-read game files" - use it after a game update.
 - The files it reads are checked against the game version this tool was made for. If your game
   files are different, it says so instead of building a broken patch.

WHAT'S NEW IN 1.0.0
 - First full release (same features as development build 2.9.2 - only the number was reset).
   Your settings, presets and recent builds carry over unchanged.
 - New icon: a map pin with a dice, on the exe and in the map page's browser tab. If Explorer
   still shows the old blank icon, that's Windows' icon cache - it updates by itself or after a restart.
 - This README was rewritten for the new layout. The full development history is in
   CHANGELOG_Map.txt.


QUICK START
 1. Double-click HZD_EnemyRandomizer_Map.exe. A small window opens and the map opens in your
    web browser. Keep the small window open while you use the map. Nothing goes online - the
    page runs on your own PC only (127.0.0.1).
 2. First time: pick your Horizon Zero Dawn folder when asked (Browse, or "Find it for me").
    The program then reads the map and machines from your game - you'll see a progress bar.
    This only happens once (they're kept in EnemyRandomizer_gamedata.cache next to the program).
 3. Choose what to change (see THE PANEL below). The quickest way: World tab > tick
    "Randomize the whole map at once".
 4. Optional: press 🔍 Preview to see how many of each machine you'll get.
 5. Press "Build & install", then start (or restart) the game.
 To undo everything: Setup tab > Uninstall.
 Make a manual save before trying risky options (anything marked ⚠).


THE MAP
 - Every machine spot in the world is shown with the game's own icons, coloured by size group
   (green Small, yellow Medium, red Large, blue Flyer, grey never changed). 920 spots in total.
 - Scroll to zoom, drag to move. "Fit map" shows everything.
 - Find a machine: type its name in the find box to highlight where it is.
 - 👤 Humans shows bandits, Eclipse and Frozen Wilds bandits (diamonds in their faction colour).
 - Regions shows the ready-made region boxes; click a region's name to switch it on or off.
 - Hover a marker to see what it is and which area it belongs to.
 - Click a marker to choose what goes there yourself:
     Randomize as normal - follows your settings (default)
     Keep it as it is    - that spot never changes
     a machine           - that machine goes there, whatever else is set
   Picked spots get a small gold dot (grey = kept).
 - Dashed markers are spots that won't change because their tick is off (Cauldrons, story
   missions, Hunting Grounds). Dotted gold rings are boss fights.
 - After a build, "Show spoilers" puts a gold ring on every spot that changed and shows what it
   became. Spoilers are hidden by default.
 - Key ▸ / ▾ folds the map key away.


THE PANEL
 Four tabs on the right. Under each tab name is a short status (e.g. "Whole map", "⚠ 2 risky",
 "Swap camps"). Build & install, 🔍 Preview and 🎲 Surprise me are always at the bottom.
 Orange tags above them list every risky option that's switched on - click one to jump to it.

 🗺 WORLD
  Whole map     one setting for every machine spot at once (your areas are kept but paused).
  Areas         draw your own boxes (Draw area, or Shift+drag) or click a ready-made region:
                Sunfall & the west, The Jewel & the south, Meridian & the Mesa,
                Carja Sundom (Daytower), The Claim & Free Heap, The Embrace,
                Nora Sacred Land, Banuk lands, The Cut (Frozen Wilds).
                Each area has an on/off switch. Where areas overlap, the newest wins.
                "Rest of the world" covers everything outside your areas (Off by default).
  Area settings (and the same for Whole map):
    Mode        Off / Grouped (machines of a similar size) / Chaotic (anything)
    Seed        same seed = same result; 🎲 rolls a new layout
    Swap chance how many spots in the area change (0-100%)
    Difficulty  Much easier ... Much harder - pushes swaps toward smaller or bigger machines
    Corrupted   chance (0-100%) that a normal spot gets the corrupted version of its machine
                (the Daemonic version in The Cut)
  Roaming herds & patrols
                one setting for all wandering herds, patrols, convoys and random encounters.
                Every herd member gets its own roll, so herds come out mixed.
    Herd size   25% - 300%. Bigger machines replacing smaller ones are capped (at 100%: up to
                6 Medium, 1 Large or 3 flyers per herd). Above 150% fights get much harder and
                the game may slow down.

 ⚔ MACHINES
  Which fights (each asks first)
    Cauldrons        machines inside SIGMA, RHO, XI, ZETA and EPSILON. Scripted doors, lifts or
                     the core override may wait for a machine that's no longer there.
    Story missions   main, side and tribe quest fights, including Frozen Wilds quests.
                     A quest can stall if it waits for a particular machine.
    Hunting Grounds  all Hunting Grounds, including the Frozen Wilds one. Trials that need a
                     particular machine may become impossible - finish the ones you care about first.
  Bosses (asks first)
    19 boss machines: main-quest Deathbringers and Corruptors, Rost's Sawtooth, the Frozen Wilds
    Frostclaws, the Fireclaw in Cauldron EPSILON and the Rockbreaker quest.
    Becomes: Big machine (default) / Medium+ / Anything. Has its own seed.
    Scripted bosses may not finish with another machine - SAVE BEFORE EACH BOSS.
  Extra machines (each asks first; each joins the Large group)
    Rockbreaker      may struggle to burrow away from its quarries.
    Corruptor        corrupts the machines around it.
    Deathbringer     experimental - reuses a main-quest Deathbringer; may stand idle or act oddly.
  Frozen Wilds & flyers
    Scorcher, Frostclaw, Fireclaw and Freeze Bellowback only appear in The Cut unless you tick
    "Anywhere" for Grouped and/or Chaotic. "Chaotic: keep flyers separate" stops flyers and
    ground machines taking each other's spots.
  Machine toughness
    Health and Damage (25% - 300%) per size group, on top of your game difficulty.
    Health 200% = they take half damage; Damage 50% = they hurt you half as much.
    Fine-tune single machines with the HP / DMG sliders in the machine list on the left.
    Applies to every machine on a map spot (Cauldrons, quests and bosses too), not roaming herds.
  Spots picked by hand: how many you've picked, and "Clear picked spots".

 👤 HUMANS
  Human enemies   bandits, Eclipse and Frozen Wilds bandits (578 spots). Friendly people are
                  never touched. Mode:
                    Same faction - types swap within their faction (safest)
                    Swap camps   - each encounter switches faction; archers stay archers
                    Chaos        - any type anywhere; factions may fight each other (asks first)
                  Extra ticks: bandit camps may change faction too (asks first); Frozen Wilds
                  bandits outside The Cut.
  Humans ⇄ machines (experimental, asks first)
                  some machine spots get a human fighter and some human spots get a machine
                  (two sliders, 0-50%). Single spots only - never herds, groups or bosses.
                  Bandit camps only if you tick it. People may wander or stand idle at machine
                  spots, and machines may get stuck in human areas.
  Human toughness Health and Damage per faction, plus "Fine-tune each type". Works even with
                  the human Mode set to Off.
  How often each type is picked - Rare / Normal / Common for each human type.

 ⚙ SETUP
  Game folder     where the game is (changeable any time - handy on another PC or drive).
  Recent builds   your last 10 installs; Restore puts that exact setup back.
  Presets         save setups by name; "Copy code" gives a short code to share, and a pasted
                  code loads someone else's setup.
  Uninstall       removes the randomizer from the game.

 MACHINE LIST (left)
  Every machine, with how many map spots it has. For each one:
    on/off switch      switched off = never appears; its spots get a similar-size machine instead
    Rare/Normal/Common how often it's picked when spots are swapped
    HP / DMG sliders   its own toughness (multiplies with its size group's sliders)


SIZE GROUPS
 Small:   Grazer, Watcher, Redeye Watcher, Strider, Lancehorn, Scrapper
 Medium:  Broadhead, Charger, Sawtooth, Ravager, Snapmaw, Trampler, Longleg, Stalker,
          Fire Bellowback, Freeze Bellowback, Shell-Walker, Scorcher
 Large:   Behemoth, Thunderjaw, Frostclaw, Fireclaw
          (+ Rockbreaker, Corruptor, Deathbringer when their ticks are on)
 Flyer:   Glinthawk, Stormbird
 Never changed: Tallneck
 Corrupted spots only ever get machines that have a corrupted version.


GOOD TO KNOW
 - Make a manual save before trying anything marked ⚠. If something gets stuck: untick it,
   Build & install again, then load your save.
 - Same settings + same seed = same result, so you can always get a layout back.
 - Machines placed by the randomizer only appear once the area reloads - start the game after
   installing, or travel away and back.
 - Use this map tool OR the old HZD_EnemyRandomizer.exe, not both: building here removes the
   other one's patch.
 - Settings are saved in EnemyRandomizer_Map_settings.json next to the program.
 - To build the program yourself: see BUILDING.txt inside source_code_map.zip (needs only Go).
 - The program contains no game files - it reads them from your own copy of the game, so it's
   fine to share the program. Don't share EnemyRandomizer_gamedata.cache (that's game data).
 - Needs Horizon Zero Dawn Complete Edition, the original PC version (not Remastered), with the
   latest update. If it says a game file is different: verify the game files in your launcher,
   then Setup > Re-read game files.
 - Deleting EnemyRandomizer_gamedata.cache is safe - the program reads the game again next time.
 - Known limits: machine size can't be changed (tested - the game ignores it), and machines
   can't be added from Horizon Forbidden West.


ANTIVIRUS FALSE ALARMS
 This program is not signed with a paid code-signing certificate, so Windows Defender sometimes
 guesses it is unsafe (detections ending in "!ml" are machine-learning guesses, not a known virus).
 What it actually does: opens a page on 127.0.0.1 (your own PC only), reads/writes its settings
 and cache files next to the exe, reads the game's archives (using the game's own
 oo2core_*_win64.dll to unpack them), and writes/removes Patch_EnemyRandomizer*.bin in the game folder.
 It doesn't use the internet or start scripts or hidden programs; it only opens your web browser.
 If Defender removes it: Windows Security > Virus & threat protection > Protection history >
 select the item > Actions > Restore (or Allow on device). You can also report the false
 positive at https://www.microsoft.com/wdsi/filesubmission so it stops being flagged.
 SHA-256 of this release (to check the file is the one built for you):
   efddfce15804765c7744fcdb24089c2b4276ad3d7de0d64fa2b6a835583dbe2f  HZD_EnemyRandomizer_Map.exe


IDEAS FOR LATER
 - Moving spawns around the map: let a machine spot be dragged to a new place on the map, or
   scatter spots randomly, instead of only changing which machine appears there.
   Not built yet. What it would involve:
     * Each spot's position is stored in the game's scene files, so it can be changed the same
       way the machine is changed now.
     * Many positions are relative to the scene they sit in, so the tool has to convert the
       map position back into that scene's own coordinates.
     * The map is flat: height isn't known, so a moved machine could start underground or in
       the air. The game usually drops it onto the ground, but not always.
     * Machines only load near their own map tile, so spots can only move a short way (within
       about the same tile) without risking them never appearing.
     * Water, cliffs and buildings: a machine moved off its walkable ground may get stuck.
   Likely first version: a "Shuffle positions" option that moves spots a small random distance
   (e.g. up to 50 m) within their own tile, with a warning.
