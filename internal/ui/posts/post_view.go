package posts

import (
	"fmt"

	"github.com/neurosnap/lists.sh/internal/db"
	"github.com/neurosnap/lists.sh/internal/ui/common"
)

type styledKey struct {
	styles    common.Styles
	date      string
	gutter    string
	postLabel string
	dateLabel string
	dateVal   string
	title     string
}

func (m Model) newStyledKey(styles common.Styles, post *db.Post) styledKey {
	publishAt := post.PublishAt
	// Default state
	return styledKey{
		styles:    styles,
		gutter:    " ",
		postLabel: "Post:",
		date:      publishAt.String(),
		dateLabel: "Added:",
		dateVal:   styles.LabelDim.Render(publishAt.String()),
		title:     post.Title,
	}
}

// Selected state
func (k *styledKey) selected() {
	k.gutter = common.VerticalLine(common.StateSelected)
	k.postLabel = k.styles.Label.Render("Post:")
	k.dateLabel = k.styles.Label.Render("Added:")
}

// Deleting state
func (k *styledKey) deleting() {
	k.gutter = common.VerticalLine(common.StateDeleting)
	k.postLabel = k.styles.Delete.Render("Post:")
	k.dateLabel = k.styles.Delete.Render("Added:")
	k.dateVal = k.styles.DeleteDim.Render(k.date)
}

func (k styledKey) render(state postState) string {
	switch state {
	case postSelected:
		k.selected()
	case postDeleting:
		k.deleting()
	}
	return fmt.Sprintf(
		"%s %s %s\n%s %s %s\n\n",
		k.gutter, k.postLabel, k.title,
		k.gutter, k.dateLabel, k.dateVal,
	)
}
