package copy

//Substitution represents transfer data substitution
type Substitution struct {
	Expand  bool              `description:"flag to substitute asset content with state keys"`
	Replace map[string]string `description:"replacements map, if key if found in the conent it wil be replaced with corresponding value."`
	ExpandIf    *Matcher          `description:"substitution source matcher"`
}
