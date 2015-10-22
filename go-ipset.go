package go_ipset

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	ipsetPath string
	errIpsetNotFound = errors.New("Ipset utility not found")
)

type Params struct {
	HashFamily string
	HashSize   int
	MaxElem    int
	Timeout    int
}

type IPSet struct {
	Name       string
	HashType   string
	HashFamily string
	HashSize   int
	MaxElem    int
	Timeout    int
}

func initCheck() error {
	if ipsetPath == "" {
		path, err := exec.LookPath("ipset")
		if err != nil {
			return errIpsetNotFound
		}
		ipsetPath = path
	}
	return nil
}

func (s *IPSet) createHashSet(name string) error {
	out, err := exec.Command(ipsetPath, "create", name, s.HashType, "family",
		s.HashFamily, "hashsize", strconv.Itoa(s.HashSize), "maxelem",
		strconv.Itoa(s.MaxElem), "timeout", strconv.Itoa(s.Timeout), "-exist").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error creating ipset %s with type %s: %v (%s)", name, s.HashType, err, out)
	}
	out, err = exec.Command(ipsetPath, "flush", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error flushing ipset %s: %v (%s)", name, err, out)
	}
	return nil
}


func New(name string, hashtype string, p *Params) (*IPSet, error) {
	if p.HashSize == 0 {
		p.HashSize = 1024
	}

	if p.MaxElem == 0 {
		p.MaxElem = 65536
	}

	if p.HashFamily == "" {
		p.HashFamily = "inet"
	}

	if !strings.HasPrefix(hashtype, "hash:") {
		return nil, fmt.Errorf("not a hash type: %s", hashtype)
	}

	if err := initCheck(); err != nil {
		return nil, err
	}

	s := IPSet{name, hashtype, p.HashFamily, p.HashSize, p.MaxElem, p.Timeout}
	err := s.createHashSet(name)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *IPSet) Refresh(entries []string) error {
	tempName := s.Name + "-temp"
	err := s.createHashSet(tempName)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		out, err := exec.Command(ipsetPath, "add", tempName, entry, "-exist").CombinedOutput()
		if err != nil {
			fmt.Errorf("error adding entry %s to set %s: %v (%s)", entry, tempName, err, out)
		}
	}
	err = Swap(tempName, s.Name)
	if err != nil {
		return err
	}
	err = destroyIPSet(tempName)
	if err != nil {
		return err
	}
	return nil
}


func (s *IPSet) Test(entry string) (bool, error) {
	out, err := exec.Command(ipsetPath, "test", s.Name, entry).CombinedOutput()
	if err == nil {
		reg, e := regexp.Compile("NOT")
		if e == nil && reg.MatchString(string(out)) {
			return false, nil
		} else if e == nil {
			return true, nil
		} else {
			return false, fmt.Errorf("error testing entry %s: %v", entry, e)
		}
	} else {
		return false, fmt.Errorf("error testing entry %s: %v (%s)", entry, err, out)
	}
}

func (s *IPSet) Add(entry string, timeout int) error {
	out, err := exec.Command(ipsetPath, "add", s.Name, entry, "timeout", strconv.Itoa(timeout), "-exist").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error adding entry %s: %v (%s)", entry, err, out)
	}
	return nil
}


func (s *IPSet) Del(entry string) error {
	out, err := exec.Command(ipsetPath, "del", s.Name, entry, "-exist").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error deleting entry %s: %v (%s)", entry, err, out)
	}
	return nil
}


func (s *IPSet) Flush() error {
	out, err := exec.Command(ipsetPath, "flush", s.Name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error flushing set %s: %v (%s)", s.Name, err, out)
	}
	return nil
}


func (s *IPSet) Destroy() error {
	out, err := exec.Command(ipsetPath, "destroy", s.Name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error destroying set %s: %v (%s)", s.Name, err, out)
	}
	return nil
}

// Swap is used to hot swap two sets on-the-fly. Use with names of existing sets of the same type.
func Swap(from, to string) error {
	out, err := exec.Command(ipsetPath, "swap", from, to).Output()
	if err != nil {
		return fmt.Errorf("error swapping ipset %s to %s: %v (%s)", from, to, err, out)
	}
	return nil
}

func destroyIPSet(name string) error {
	out, err := exec.Command(ipsetPath, "destroy", name).Output()
	if err != nil {
		return fmt.Errorf("error destroying ipset %s: %v (%s)", name, err, out)
	}
	return nil
}
