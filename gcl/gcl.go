package main

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type mpath struct {
	name string
	path string
}

type vdir string

func (vd vdir) trunc() string {
	s := string(vd)
	if len(s) > 0 {
		return s[:7]
	}
	return ""
}

var mpathre = regexp.MustCompile(`^\.(.*)###(.*)$`)
var vdirre = regexp.MustCompile(`^[a-f0-9]{64}$`)

func newmpath(mp string) *mpath {
	res := mpathre.FindAllStringSubmatch(mp, -1)
	var mres *mpath
	if res != nil && len(res) == 1 && len(res[0]) == 3 {
		name := res[0][1]
		path := res[0][2]
		path = strings.Replace(path, ",#,", "/", -1)
		mres = &mpath{name, path}
	}
	return mres
}

func (mp *mpath) String() string {
	return "." + mp.name + "###" + strings.Replace(mp.path, "/", ",#,", -1)
}

func newvdir(vd string) vdir {
	res := vdirre.FindString(vd)
	vres := vdir("")
	if res != "" && res == vd {
		vres = vdir(vd)
	}
	return vres
}

type marker struct {
	mp  *mpath
	dir vdir
}

func (mk *marker) String() string {
	return fmt.Sprintf("marker '%s'<%s->%s>", mk.dir.trunc(), mk.mp.name, mk.mp.path)
}

type markers []*marker

var allmarkers = markers{}

func (marks markers) getMarker(name string, path string, dir vdir) *marker {
	var mark *marker
	var pathdiff *marker
	for _, marker := range marks {
		if marker.mp.name == name {
			if marker.mp.path == path {
				mark = marker
				break
			} else {
				pathdiff = marker
			}
		}
	}
	if mark == nil {
		if pathdiff != nil {
			fmt.Printf("Invalid volume path detected: '%s' (not found in marker '%s')\n", path, pathdiff)
			return nil
		}
		mark = &marker{mp: &mpath{name: name, path: path}, dir: dir}
	}
	ldir := mark.dir
	if mark.dir != dir {
		// move dir and ln -s mark.dir dir
		ldir = dir
		cmd(fmt.Sprintf("sudo move /mnt/sda1/var/lib/docker/vfs/dir/%s /mnt/sda1/var/lib/docker/vfs/dir/_%s", dir, dir))
		fmt.Printf("Mark container named '%s' for path '%s' as link to '%s' (from '%s')\n", name, path, mark.dir, dir)
	}
	_, err := cmd(fmt.Sprintf("sudo ls -1L /mnt/sda1/var/lib/docker/vfs/dir/%s", mark.mp.String()))
	if err != nil {
		mustcmd(fmt.Sprintf("sudo ln -s %s /mnt/sda1/var/lib/docker/vfs/dir/%s", ldir, mark.mp.String()))
	}
	return mark
}

type volume struct {
	dir      vdir
	mark     *marker
	orphaned bool
}

type volumes []*volume

var allvolumes = volumes{}

func (v *volume) String() string {
	return fmt.Sprintf("vol '%s'<%v>", v.dir.trunc(), v.mark)
}

func (vols volumes) getVolume(vd vdir, path string, name string) *volume {
	var vol *volume
	for _, volume := range vols {
		if string(volume.dir) == string(vd) {
			vol = volume
			if vol.mark == nil {
				vol.mark = allmarkers.getMarker(name, path, vol.dir)
			}
			if vol.mark == nil {
				return nil
			}
			vol.orphaned = false
			break
		}
	}
	if vol == nil {
		vol = &volume{dir: vd, orphaned: true}
		vol.mark = allmarkers.getMarker(name, path, vol.dir)
		if vol.mark == nil {
			return nil
		}
		vol.orphaned = false
	}
	return vol
}

type container struct {
	name     string
	id       string
	stopped  bool
	orphaned bool
	volumes  []*volume
}

func (c *container) trunc() string {
	switch {
	case len(c.id) == 0:
		return ""
	case len(c.id) > 6:
		return c.id[:7]
	default:
		return c.id
	}
}

func (c *container) String() string {
	return "cnt '" + c.name + "' (" + c.trunc() + ")" + fmt.Sprintf("[%v] - %d vol", c.stopped, len(c.volumes))
}

type containers []*container

var allcontainers = containers{}

func mustcmd(acmd string) string {
	out, err := cmd(acmd)
	if err != nil {
		log.Fatal(fmt.Sprintf("out='%s', err='%s'", out, err))
	}
	return string(out)
}

func initLists() {
	allmarkers = markers{}
	allvolumes = volumes{}
	allcontainers = containers{}
	orphanedContainers = containers{}
	orphanedVolumes = volumes{}
}

type fcmd func(cmd string) (string, error)

var cmd = execcmd

func execcmd(cmd string) (string, error) {
	fmt.Println(cmd)
	out, err := exec.Command("sh", "-c", cmd).Output()
	return strings.TrimSpace(string(out)), err
}

// Looks for volume folder or marker symlinks in /var/lib/docker/vfs/dir
func readVolumes() {
	out := mustcmd("sudo ls -a1F /mnt/sda1/var/lib/docker/vfs/dir")
	vollines := strings.Split(out, "\n")
	for _, volline := range vollines {
		dir := volline
		if dir == "./" || dir == "../" || dir == "" {
			continue
		}
		if strings.HasSuffix(dir, "@") {
			dir = dir[:len(dir)-1]
			fdir := fmt.Sprintf("/mnt/sda1/var/lib/docker/vfs/dir/%s", dir)
			mp := newmpath(dir)
			if mp == nil {
				fmt.Printf("Invalid marker detected: '%s'\n", dir)
				mustcmd("sudo rm " + fdir)
			} else {
				dirlink, err := cmd("sudo readlink " + fdir)
				fmt.Printf("---\ndir: '%s'\ndlk: '%s'\nerr='%v'\n", dir, dirlink, err)
				if err != nil {
					fmt.Printf("Invalid marker (no readlink) detected: '%s'\n", dir)
					mustcmd("sudo rm " + fdir)
				} else {
					_, err := cmd("sudo ls /mnt/sda1/var/lib/docker/vfs/dir/" + dirlink)
					if err != nil {
						fmt.Printf("Invalid marker (readlink no ls) detected: '%s'\n", dir)
						mustcmd("sudo rm " + fdir)
					} else {
						vd := newvdir(dirlink)
						if vd == "" {
							fmt.Printf("Invalid marker (readlink no vdir) detected: '%s'\n", dir)
							mustcmd("sudo rm " + fdir)
						} else {
							allmarkers = append(allmarkers, &marker{mp, vd})
						}
					}
				}
			}
		} else if strings.HasSuffix(dir, "/") {

			dir = dir[:len(dir)-1]
			fdir := fmt.Sprintf("/mnt/sda1/var/lib/docker/vfs/dir/%s", dir)
			vd := newvdir(dir)
			if vd == "" {
				fmt.Printf("Invalid volume folder detected: '%s'\n", dir)
				mustcmd("sudo rm " + fdir)
			} else {
				allvolumes = append(allvolumes, &volume{dir: vd, orphaned: true})
			}
		} else {
			fdir := fmt.Sprintf("/mnt/sda1/var/lib/docker/vfs/dir/%s", dir)
			fmt.Printf("Invalid file detected: '%s'\n", dir)
			mustcmd("sudo rm " + fdir)
		}
	}
}

func readContainer() {

	out := mustcmd("docker ps -aq --no-trunc")
	contlines := strings.Split(out, "\n")
	// fmt.Println(contlines)
	for _, contline := range contlines {
		if contline == "" {
			continue
		}
		id := contline
		res := mustcmd("docker inspect -f '{{ .Name }},{{ range $key, $value := .Volumes }}{{ $key }},{{ $value }}##~#{{ end }}' " + id)
		// fmt.Println("res1: '" + res + "'")
		name := res[1:strings.Index(res, ",")]
		cont := &container{name: name, id: id}
		res = res[strings.Index(res, ",")+1:]
		// fmt.Println("res2: '" + res + "'")
		vols := strings.Split(res, "##~#")
		// fmt.Println(vols)
		for _, vol := range vols {
			elts := strings.Split(vol, ",")
			if len(elts) == 2 {
				// fmt.Printf("elts: '%v'\n", elts)
				path := elts[0]
				vfs := elts[1]
				if strings.Contains(vfs, "/var/lib/docker/vfs/dir/") {
					vd := newvdir(filepath.Base(vfs))
					if vd == "" {
						fmt.Printf("Invalid container volume folder detected: '%s'\n", vfs)
						break
					}
					newvol := allvolumes.getVolume(vd, path, name)
					if newvol != nil {
						cont.volumes = append(cont.volumes, newvol)
					} else {
						cont.orphaned = true
					}

				}
			}
		}
		allcontainers = append(allcontainers, cont)
		if cont.orphaned {
			orphanedContainers = append(orphanedContainers, cont)
		}
	}
}

// docker run --rm -i -t -v `pwd`:`pwd` -w `pwd` --entrypoint="/bin/bash" go -c 'go build gcl.go'
func main() {
	initLists()
	readVolumes()
	readContainer()
	for _, vol := range allvolumes {
		if vol.orphaned {
			orphanedVolumes = append(orphanedVolumes, vol)
		}
	}
}

// Containers returns all detected containers
func Containers() []*container {
	return allcontainers
}

var orphanedContainers = []*container{}

// OrphanedContainers returns all detected containers with missing volumes
// They are not orphaned if they have no volumes
// They are orphaned if they have volumes which don't exist
func OrphanedContainers() []*container {
	return orphanedContainers
}

// Volumes returns all detected volumes
func Volumes() volumes {
	return allvolumes
}

var orphanedVolumes = volumes{}

// OrphanedVolumes returns all detected volumes with missing containers
// They are orphaned because no container reference them directly
// or through a marker
func OrphanedVolumes() volumes {
	return orphanedVolumes
}

// Markers returns all detected or created markers
func Markers() markers {
	return allmarkers
}
