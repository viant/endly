package docker

import (
	"fmt"
	"strings"
)

//Tag represent a docker tag
type Tag struct {
	Username string
	Registry string
	Image    string
	Version  string
}

func (t *Tag) Repository() string {
	if t.Registry == "" {
		t.Registry = "index.docker.io"
	}
	return fmt.Sprintf("%v/%v", t.Registry, t.Username)
}

//String stringify docker tag
func (t *Tag) String() string {
	result := t.Registry
	if result == "" {
		result = t.Username
	} else if t.Username != "" {
		result += "/" + t.Username
	}
	if result != "" {
		result += "/"
	}
	result += t.Image
	if t.Version != "" {
		result += ":" + t.Version
	}
	return result
}

//NewTag returns new tag
func NewTag(imageTag string) *Tag {
	tag := &Tag{}
	parts := strings.SplitN(imageTag, ":", 2)
	if len(parts) == 2 {
		tag.Version = parts[1]
		imageTag = parts[0]
	}
	tag.Image = imageTag
	if imageIndex := strings.LastIndex(imageTag, "/"); imageIndex != -1 {
		tag.Registry = string(imageTag[:imageIndex])
		tag.Image = string(imageTag[imageIndex+1:])
		if registryIndex := strings.LastIndex(tag.Username, "/"); registryIndex != -1 {
			tag.Username = string(tag.Username[registryIndex+1:])
		}

	}
	return tag
}
