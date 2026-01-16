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
	image    models.Image
	selected bool
}

func (i ImageItem) FilterValue() string {
	return i.image.GetPrimaryTag()
}

func (i ImageItem) Title() string {
	title := i.image.GetPrimaryTag()

	// Add status markers
	var markers []string
	if i.image.IsDangling() {
		markers = append(markers, styles.WarningStyle.Render("[dangling]"))
	}
	if i.image.IsUnused() {
		markers = append(markers, styles.SubtitleStyle.Render("[unused]"))
	}

	// Add selection marker
	selectMark := "  "
	if i.selected {
		selectMark = styles.SuccessStyle.Render("✓ ")
	}

	if len(markers) > 0 {
		return selectMark + title + " " + strings.Join(markers, " ")
	}
	return selectMark + title
}

func (i ImageItem) Description() string {
	size := formatBytes(i.image.Size)
	containers := ""
	if i.image.Containers > 0 {
		containers = fmt.Sprintf(" • %d container(s)", i.image.Containers)
	}
	return fmt.Sprintf("   ID: %s • Size: %s%s", i.image.ShortID, size, containers)
}

// ImagesView displays the list of images
type ImagesView struct {
	list     list.Model
	images   []models.Image
	selected map[string]bool // Map of image ID to selection state
	width    int
	height   int
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
		list:     l,
		selected: make(map[string]bool),
	}
}

// SetImages updates the list of images
func (v *ImagesView) SetImages(images []models.Image) {
	v.images = images

	// Clean up selected map - remove IDs that no longer exist
	existingIDs := make(map[string]bool)
	for _, img := range images {
		existingIDs[img.ID] = true
	}
	for id := range v.selected {
		if !existingIDs[id] {
			delete(v.selected, id)
		}
	}

	v.rebuildList()
}

// rebuildList rebuilds the list items with current selection state
func (v *ImagesView) rebuildList() {
	items := make([]list.Item, len(v.images))
	for i, img := range v.images {
		items[i] = ImageItem{
			image:    img,
			selected: v.selected[img.ID],
		}
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
	item := v.list.SelectedItem()
	if item == nil {
		return nil
	}
	if imageItem, ok := item.(ImageItem); ok {
		return &imageItem.image
	}
	return nil
}

// ToggleSelection toggles selection of the current image
func (v *ImagesView) ToggleSelection() {
	img := v.GetSelectedImage()
	if img == nil {
		return
	}

	if v.selected[img.ID] {
		delete(v.selected, img.ID)
	} else {
		v.selected[img.ID] = true
	}
	v.rebuildList()
}

// GetSelectedImages returns all selected images
func (v *ImagesView) GetSelectedImages() []models.Image {
	var result []models.Image
	for _, img := range v.images {
		if v.selected[img.ID] {
			result = append(result, img)
		}
	}
	return result
}

// HasSelection returns true if any images are selected
func (v *ImagesView) HasSelection() bool {
	return len(v.selected) > 0
}

// ClearSelection clears all selections
func (v *ImagesView) ClearSelection() {
	v.selected = make(map[string]bool)
	v.rebuildList()
}

// GetSelectionCount returns the number of selected images
func (v *ImagesView) GetSelectionCount() int {
	return len(v.selected)
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
		styles.KeyStyle.Render("space") + " select",
		styles.KeyStyle.Render("d") + " remove",
		styles.KeyStyle.Render("p") + " pull",
		styles.KeyStyle.Render("P") + " prune",
		styles.KeyStyle.Render("/") + " filter",
		styles.KeyStyle.Render("q") + " quit",
	}

	// Show selection count if any
	if v.HasSelection() {
		helps = append([]string{styles.SuccessStyle.Render(fmt.Sprintf("[%d selected]", v.GetSelectionCount()))}, helps...)
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
