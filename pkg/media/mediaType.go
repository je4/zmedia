package media

type MediaType interface {
	MimeOK(mimetype string) bool
}

type CoreMeta struct {
	Width    int64
	Height   int64
	Duration int64
	Format   string
	Mimetype string
}
