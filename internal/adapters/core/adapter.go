package core

type Adapter interface {
	ToTemplateData() (*TemplateData, error)
	GetSourceType() string
}
