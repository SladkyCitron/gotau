package gotau

const (
	// ResamplerDiskCacheDir is the name of the subdirectory in the user's cache directory where
	// resampled notes will be cached when using diskcache.
	ResamplerDiskCacheDir = "gotau/resampler"

	// ResamplerDiskCacheExt is the file extension used for cached resampled notes when using diskcache.
	ResamplerDiskCacheExt = ".wav"
)

// Progress represents the rendering progress information.
type Progress struct{}
