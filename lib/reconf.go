package lib

import (
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/apex/log"
)

// Reconf writes ssh config with profiles into the specified file
func Reconf(profiles []ProfileConfig, filename string, noProfilePrefix bool) {
	profileSummaries, err := TraverseProfiles(profiles, noProfilePrefix)
	if err != nil {
		log.WithError(err).Error("got some errors")
		return
	}

	var sshEntries []SSHEntry

	// go through all profileSummaries and
	// create sshEntries out of it
	for _, summary := range profileSummaries {
		sshEntries = append(sshEntries, summary.SSHEntries...)
	}

	tmpfile, err := ioutil.TempFile(path.Dir(filename), "aws-ssh")
	ctx := log.WithField("tmpfile", tmpfile.Name())
	if err != nil {
		ctx.WithError(err).Fatal("Couldn't create a temporary file")
	}

	for _, entry := range sshEntries {
		if _, err := io.WriteString(tmpfile, entry.ConfigFormat()); err != nil {
			ctx.WithError(err).Fatal("Can't write to the temp file")
		}
	}
	if err := tmpfile.Close(); err != nil {
		ctx.WithError(err).Fatal("Couldn't close the temporary file")
	}
	if err := os.Rename(tmpfile.Name(), filename); err != nil {
		ctx.WithError(err).Fatalf("Couldn't move the file %s to %s", tmpfile.Name(), filename)
	}
}
