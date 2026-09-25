package golang

type OutputFile string

const (
	OutputFileModel    OutputFile = "modelFile"
	OutputFileQuery    OutputFile = "queryFile"
	OutputFileDb       OutputFile = "dbFile"
	OutputFileCopyfrom OutputFile = "copyfromFile"
	OutputFileBatch    OutputFile = "batchFile"
)
