package tui

func (m *Model) resize() {
	tableHeight := m.tableHeight()
	if m.table.Height() != tableHeight-1 {
		m.table.SetHeight(tableHeight)
	}
	m.viewport.Width = maxInt(20, m.width-8)
	m.viewport.Height = maxInt(4, m.height-tableHeight-12)
	if m.details {
		m.viewport.Height = maxInt(3, m.viewport.Height-m.detailHeight())
	}
}

func (m *Model) tableHeight() int {
	if m.height <= 0 {
		return maxInt(6, m.table.Height())
	}
	return maxInt(6, m.height/2-6)
}

func (m *Model) detailHeight() int {
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return 3
	}
	return minInt(12, len(row)+5)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
