package storage

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/afs/storage"
	arl "github.com/viant/afs/url"
	"github.com/viant/toolbox/url"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"path"
)

func (s *service) compressSource(context *endly.Context, source, target *url.Resource, sourceObject storage.Object) (err error) {
	var baseDirectory, name = path.Split(source.ParsedURL.Path)
	var archiveSource = name

	if sourceObject.IsDir() {
		baseDirectory = source.DirectoryPath()
		_, name = path.Split(baseDirectory)
		archiveSource = "."
	}
	var archiveName = fmt.Sprintf("%v.tar.gz", name)

	if source.ParsedURL.Scheme == "file" && source.Credentials == "" {
		source.Credentials = "localhost"
	}
	var runRequest = exec.NewRunRequest(source, false,
		fmt.Sprintf("cd %v", baseDirectory),
		fmt.Sprintf("tar cvzf %v %v", archiveName, archiveSource),
	)
	runRequest.TimeoutMs = compressionTimeoutMs
	runResponse := &exec.RunResponse{}
	if err = endly.Run(context, runRequest, runResponse); err != nil {
		return err
	}
	if util.CheckNoSuchFileOrDirectory(runResponse.Stdout()) {
		return fmt.Errorf("faied to compress: %v, %v", fmt.Sprintf("tar cvzf %v %v", archiveName, archiveSource), runResponse.Stdout())
	}

	if sourceObject.IsDir() {
		source.URL = arl.Join(source.URL, archiveName)
		_ = source.Init()

		target.URL = arl.Join(target.URL, archiveName)
		_ = target.Init()
		return nil
	}

	if err = source.Rename(archiveName); err == nil {
		if path.Ext(target.ParsedURL.Path) != "" {
			_, targetName := path.Split(target.ParsedURL.Path)
			if name != targetName {
				err = target.Rename(fmt.Sprintf("%v.tar.gz", targetName))
			} else {
				err = target.Rename(archiveName)
			}
		} else {
			target.URL = arl.Join(target.URL, archiveName)
			_ = target.Init()
		}
	}
	return err
}



func (s *service) decompressTarget(context *endly.Context, source, target *url.Resource, sourceObject storage.Object) error {
	var baseDir, name = path.Split(target.ParsedURL.Path)
	var runRequest = exec.NewRunRequest(target, false,
		fmt.Sprintf("mkdir -p %v", baseDir),
		fmt.Sprintf("cd %v", baseDir),
		fmt.Sprintf("tar xvzf %v", name),
		fmt.Sprintf("rm %v", name),
		fmt.Sprintf("cd %v", source.DirectoryPath()))
	runRequest.TimeoutMs = compressionTimeoutMs
	return endly.Run(context, runRequest, nil)
}

