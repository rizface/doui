package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rizface/doui/internal/models"
	"github.com/rizface/doui/internal/ui/styles"
)

// ImageItem implements list.Item for images
type ImageItem struct {
	image models.Image
}

func (i ImageItem) FilterValue() string {
	return i.image.GetPrimaryTag()
}

func (i ImageItem) Title() string {
	return i.image.GetPrimaryTag()
}

func (i ImageItem) Description() string {
	size := formatBytes(i.image.Size)
	containers := ""
	if i.image.Containers > 0 {
		containers = fmt.Sprintf(" • %d container(s)", i.image.Containers)
	}
	return fmt.Sprintf("ID: %s • Size: %s%s", i.image.ShortID, size, containers)
}

// ImagesView displays the list of images
type ImagesView struct {
	list   list.Model
	images []models.Image
	width  int
	height int
}

// NewImagesView creates a new images view
func NewImagesView() *ImagesView {
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(2)
	delegate.SetSpacing(1)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Docker Images"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = styles.TitleStyle

	return &ImagesView{
		list: l,
	}
}

// SetImages updates the list of images
func (v *ImagesView) SetImages(images []models.Image) {
	v.images = images

	items := make([]list.Item, len(images))
	for i, img := range images {
		items[i] = ImageItem{image: img}
	}

	v.list.SetItems(items)
}

// SetSize updates the view dimensions
func (v *ImagesView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.list.SetSize(width, height-6)
}

// Update handles messages
func (v *ImagesView) Update(msg tea.Msg) (*ImagesView, tea.Cmd) {
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

// View renders the view
func (v *ImagesView) View() string {
	if len(v.images) == 0 {
		return v.renderEmpty()
	}

	return v.list.View()
}

// GetSelectedImage returns the currently selected image
func (v *ImagesView) GetSelectedImage() *models.Image {
	if len(v.images) == 0 || v.list.Index() >= len(v.images) {
		return nil
	}
	return &v.images[v.list.Index()]
}

func (v *ImagesView) renderEmpty() string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Docker Images"))
	b.WriteString("\n\n")
	b.WriteString(styles.SubtitleStyle.Render("No images found. Pull some Docker images to see them here."))

	return b.String()
}

// IsFiltering returns true if the list is in filtering mode
func (v *ImagesView) IsFiltering() bool {
	return v.list.FilterState() == list.Filtering
}

// GetHelpText returns help text for the images view
func (v *ImagesView) GetHelpText() string {
	helps := []string{
		styles.KeyStyle.Render("↑/↓") + " navigate",
		styles.KeyStyle.Render("d") + " remove",
		styles.KeyStyle.Render("/") + " filter",
		styles.KeyStyle.Render("q") + " quit",
	}

	return strings.Join(helps, styles.SeparatorStyle.String())
}

// formatBytes formats bytes to human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
