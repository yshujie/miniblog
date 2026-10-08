package catalog

type ModuleInput struct {
	Code, Title string
	Sort        *int
}
type SectionInput struct {
	Code, Title, ModuleCode string
	Sort                    *int
}
type SubsectionInput struct {
	Code, Title, SectionCode string
	Sort                     *int
}
type UpdateInput struct {
	Title string
	Sort  *int
}
type ReorderInput struct {
	Kind, ParentCode string
	Codes            []string
}
