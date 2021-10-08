package cache

import (
	"aws-ssh/lib"
	"fmt"
	"os"
	"path"
	"sort"
	"time"

	"gopkg.in/yaml.v2"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

const instancesDir = "instances"

type YAMLCache struct {
	basedir string
	index   YAMLCacheIndex
}

type YAMLCacheIndex struct {
	Time           time.Time
	InstancesIndex map[string]string
	CanonicalNames []string
}

func (y *YAMLCache) saveIndex() error {
	// save index
	var indexFileName = path.Join(y.basedir, fmt.Sprintf("index.yaml"))

	indexFile, err := os.OpenFile(indexFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("can't open %s: %s", indexFileName, err)
	}

	defer indexFile.Close()
	encoder := yaml.NewEncoder(indexFile)
	err = encoder.Encode(&y.index)
	if err != nil {
		return fmt.Errorf("can't encode %s: %s", indexFileName, err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("can't close %s: %s", indexFileName, err)
	}
	return nil
}

func (y *YAMLCache) loadIndex() error {
	// the index has been loaded
	if !y.index.Time.IsZero() {
		return nil
	}
	// load index
	var indexFileName = path.Join(y.basedir, fmt.Sprintf("index.yaml"))
	indexFile, err := os.OpenFile(indexFileName, os.O_RDONLY, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cache doesn't exist, try \"aws-ssh update\"")
		}
		return fmt.Errorf("can't open %s: %s", indexFileName, err)
	}

	defer indexFile.Close()
	decoder := yaml.NewDecoder(indexFile)
	err = decoder.Decode(&y.index)
	if err != nil {
		return fmt.Errorf("can't decode %s: %s", indexFileName, err)
	}
	return nil
}

func (y *YAMLCache) Load() ([]lib.ProcessedProfileSummary, error) { return nil, nil }
func (y *YAMLCache) Save(profileSummaries []lib.ProcessedProfileSummary) error {
	var instancesPath = path.Join(y.basedir, instancesDir)
	// map of all aliases -> instance id
	var index = make(map[string]string)

	err := os.MkdirAll(instancesPath, 0700)
	if err != nil {
		return err
	}
	// every ssh entry is self-contained
	for _, summary := range profileSummaries {
		for _, sshEntry := range summary.SSHEntries {
			var fileName = path.Join(instancesPath, fmt.Sprintf("%s.yaml", sshEntry.InstanceID))
			file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return fmt.Errorf("can't open %s: %s", fileName, err)
			}

			defer file.Close()
			encoder := yaml.NewEncoder(file)
			err = encoder.Encode(&sshEntry)
			if err != nil {
				return fmt.Errorf("can't encode %s: %s", fileName, err)
			}
			if err := encoder.Close(); err != nil {
				return fmt.Errorf("can't close %s: %s", fileName, err)
			}

			// add every instance name to the index and resolve to instance id
			for n, name := range sshEntry.Names {
				if name != sshEntry.InstanceID {
					index[name] = sshEntry.InstanceID
					// add the first name to canonical names
					if n == 0 {
						y.index.CanonicalNames = append(y.index.CanonicalNames, name)
					}
				} else {
					index[name] = ""
				}
			}
		}
	}
	sort.Strings(y.index.CanonicalNames)
	y.index.InstancesIndex = index

	return y.saveIndex()
}
func (y *YAMLCache) Lookup(name string) (lib.SSHEntry, error) {
	var entry lib.SSHEntry
	err := y.loadIndex()
	if err != nil {
		return entry, err
	}
	var instanceID string
	// switch to fuzzy match if don't have exact match
	// or name was no provided
	if val, ok := y.index.InstancesIndex[name]; !ok || name == "" {
		if len(y.index.CanonicalNames) == 0 {
			return entry, fmt.Errorf("no names in index, try \"aws-ssh update\"")
		}
		idx, err := fuzzyfinder.Find(y.index.CanonicalNames, func(i int) string {
			return fmt.Sprintf("%s", y.index.CanonicalNames[i])
		})
		if err == fuzzyfinder.ErrAbort {
			return entry, fmt.Errorf("nothing was selected in fuzzy match")
		}
		instanceID = y.index.InstancesIndex[y.index.CanonicalNames[idx]] // The selected item.
	} else {
		// if key is instanceid then value will be empty
		if val == "" {
			instanceID = name
		} else {
			instanceID = y.index.InstancesIndex[name]
		}
	}

	var fileName = path.Join(
		y.basedir,
		instancesDir,
		fmt.Sprintf("%s.yaml", instanceID),
	)
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return entry, fmt.Errorf("can't open %s: %s", fileName, err)
	}

	defer file.Close()
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&entry)
	if err != nil {
		return entry, fmt.Errorf("can't decode %s: %s", fileName, err)
	}

	return entry, nil
}

func (y *YAMLCache) ListCanonicalNames() ([]string, error) {
	if err := y.loadIndex(); err != nil {
		return []string{}, nil
	}
	return y.index.CanonicalNames, nil
}

func NewYAMLCache(basedir string) Cache {

	return &YAMLCache{basedir: basedir}
}
