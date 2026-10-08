package main

import (
	"image"
	_ "image/png"
	"os"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

// usage: wr out.syso "File description" "Product name" "1.4.4.0" "OriginalName.exe"
func main() {
	rs := winres.ResourceSet{}
	vi := version.Info{}
	vi.SetFileVersion(os.Args[4])
	vi.SetProductVersion(os.Args[4])
	vi.Set(0, version.FileDescription, os.Args[2])
	vi.Set(0, version.ProductName, os.Args[3])
	vi.Set(0, version.OriginalFilename, os.Args[5])
	vi.Set(0, version.InternalName, os.Args[5])
	vi.Set(0, version.CompanyName, "HZD Enemy Randomizer (fan-made mod tool)")
	vi.Set(0, version.LegalCopyright, "Free fan-made tool. Not affiliated with Guerrilla Games or Sony.")
	vi.Set(0, version.Comments, "Builds a randomizer patch file for Horizon Zero Dawn Complete Edition. Runs locally; no network access except a local web page on 127.0.0.1.")
	rs.SetVersionInfo(vi)
	rs.SetManifest(winres.AppManifest{
		Identity:            winres.AssemblyIdentity{Name: "HZD.EnemyRandomizer", Version: [4]uint16{1, 4, 4, 0}},
		Description:         os.Args[2],
		ExecutionLevel:      winres.AsInvoker,
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
	})
	if len(os.Args) > 6 { // optional: PNG icon (256x256 or bigger)
		pf, err := os.Open(os.Args[6])
		if err != nil {
			panic(err)
		}
		img, _, err := image.Decode(pf)
		pf.Close()
		if err != nil {
			panic(err)
		}
		icon, err := winres.NewIconFromResizedImage(img, nil)
		if err != nil {
			panic(err)
		}
		if err := rs.SetIcon(winres.Name("APPICON"), icon); err != nil {
			panic(err)
		}
	}
	f, err := os.Create(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := rs.WriteObject(f, winres.ArchAMD64); err != nil {
		panic(err)
	}
}
