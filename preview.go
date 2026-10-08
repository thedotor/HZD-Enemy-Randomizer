package main

// previewMap: what a build would put in the world, counted per machine, without making the patch.
func previewMap(c MapConfig) map[string]interface{} {
	c.planOnly = true
	c.tally = map[string]int{}
	_, changes, err := buildMapPatch(c)
	if err != nil {
		return map[string]interface{}{"ok": false, "message": err.Error()}
	}
	final := map[int]string{}
	humans, toHumans, toMachines := 0, 0, map[string]int{}
	for _, ch := range changes {
		if ch.Cross {
			if ch.Human {
				toMachines[ch.To]++
			} else {
				toHumans++
				final[ch.ID] = ""
			}
			continue
		}
		if ch.Human {
			humans++
			continue
		}
		final[ch.ID] = ch.To
	}
	spots := map[string]int{}
	changed := 0
	for _, s := range mapData.Sites {
		m := s.Machine
		if to, ok := final[s.ID]; ok {
			m = to
			changed++
		}
		if m != "" {
			spots[m]++
		}
	}
	for m, n := range toMachines {
		spots[m] += n
	}
	return map[string]interface{}{"ok": true, "spots": spots, "herds": c.tally, "changed": changed, "humans": humans, "to_humans": toHumans, "to_machines": sumCounts(toMachines),
		"total_spots": len(mapData.Sites)}
}

func sumCounts(m map[string]int) int {
	t := 0
	for _, n := range m {
		t += n
	}
	return t
}
