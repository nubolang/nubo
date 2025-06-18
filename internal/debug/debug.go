package debug

type Debug struct {
	// Line represents the line of the node
	Line int `yaml:"line"`
	// Column represents the column of the node
	Column int `yaml:"column"`
	// ColumnEnd represents the end index
	ColumnEnd int `yaml:"column_end"`
	// File represents the file of the node
	File string `yaml:"file"`
	// Near represents the code near the node
	Near string `yaml:"-"`
}

func (d *Debug) GetDebug() *Debug {
	return d
}
