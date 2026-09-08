package webdriver

import (
	"fmt"
	"strings"
	"time"

	"github.com/tebeka/selenium"
)

// Tabs returns every browser tab/window and marks the currently selected one.
func (s *service) Tabs(driver selenium.WebDriver) ([]map[string]interface{}, error) {
	handles, err := driver.WindowHandles()
	if err != nil {
		return nil, err
	}
	current, _ := driver.CurrentWindowHandle()
	result := make([]map[string]interface{}, 0, len(handles))
	for _, handle := range handles {
		if err := driver.SwitchWindow(handle); err != nil {
			return nil, fmt.Errorf("switch to tab %s: %w", handle, err)
		}
		URL, _ := driver.CurrentURL()
		title, _ := driver.Title()
		result = append(result, map[string]interface{}{
			"handle":  handle,
			"url":     URL,
			"title":   title,
			"current": handle == current,
		})
	}
	if current != "" {
		_ = driver.SwitchWindow(current)
	}
	return result, nil
}

// SwitchTab accepts an exact window handle or a case-insensitive URL/title
// fragment. The original tab remains selected when no match is found.
func (s *service) SwitchTab(driver selenium.WebDriver, target string) (map[string]interface{}, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("tab target was empty")
	}
	handles, err := driver.WindowHandles()
	if err != nil {
		return nil, err
	}
	original, _ := driver.CurrentWindowHandle()
	for _, handle := range handles {
		if handle == target {
			if err := driver.SwitchWindow(handle); err != nil {
				return nil, err
			}
			return currentTab(driver, handle)
		}
	}
	lowerTarget := strings.ToLower(target)
	for _, handle := range handles {
		if err := driver.SwitchWindow(handle); err != nil {
			continue
		}
		URL, _ := driver.CurrentURL()
		title, _ := driver.Title()
		if strings.Contains(strings.ToLower(URL), lowerTarget) || strings.Contains(strings.ToLower(title), lowerTarget) {
			return map[string]interface{}{"handle": handle, "url": URL, "title": title, "current": true}, nil
		}
	}
	if original != "" {
		_ = driver.SwitchWindow(original)
	}
	return nil, fmt.Errorf("tab matching %q was not found", target)
}

// NewTab opens a tab, switches to it, and optionally navigates to URL.
func (s *service) NewTab(driver selenium.WebDriver, URL string) (map[string]interface{}, error) {
	if direct, ok := driver.(interface {
		OpenTab(string) (string, error)
	}); ok {
		handle, err := direct.OpenTab(URL)
		if err != nil {
			return nil, err
		}
		return currentTab(driver, handle)
	}
	before, err := driver.WindowHandles()
	if err != nil {
		return nil, err
	}
	if _, err = driver.ExecuteScript("window.open('about:blank', '_blank');", nil); err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(before))
	for _, handle := range before {
		known[handle] = true
	}
	deadline := time.Now().Add(2 * time.Second)
	var handle string
	for time.Now().Before(deadline) {
		handles, handlesErr := driver.WindowHandles()
		if handlesErr != nil {
			return nil, handlesErr
		}
		for _, candidate := range handles {
			if !known[candidate] {
				handle = candidate
				break
			}
		}
		if handle != "" {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if handle == "" {
		return nil, fmt.Errorf("new tab did not appear")
	}
	if err := driver.SwitchWindow(handle); err != nil {
		return nil, err
	}
	if strings.TrimSpace(URL) != "" {
		if err := driver.Get(URL); err != nil {
			return nil, err
		}
	}
	return currentTab(driver, handle)
}

// CloseTab closes the current tab and selects a remaining tab when available.
func (s *service) CloseTab(driver selenium.WebDriver) (map[string]interface{}, error) {
	if err := driver.Close(); err != nil {
		return nil, err
	}
	handles, err := driver.WindowHandles()
	if err != nil {
		return nil, err
	}
	if len(handles) == 0 {
		return map[string]interface{}{}, nil
	}
	handle := handles[len(handles)-1]
	if err := driver.SwitchWindow(handle); err != nil {
		return nil, err
	}
	return currentTab(driver, handle)
}

func currentTab(driver selenium.WebDriver, handle string) (map[string]interface{}, error) {
	URL, err := driver.CurrentURL()
	if err != nil {
		return nil, err
	}
	title, _ := driver.Title()
	return map[string]interface{}{"handle": handle, "url": URL, "title": title, "current": true}, nil
}
