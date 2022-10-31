package util

func ForestInsert(f *Forest, cid []string, firstIsFile bool) *Entry {
	newRoot := cid[0]
	if len(cid) == 1 {
		cid = nil
	} else if len(cid) > 1 {
		cid = cid[1:]
	}
	for _, root := range f.Trees {
		if newRoot == root.Cid {
			return InsertWithFullPath(root, cid)
		}
	}
	typ := EntryTyp_Dir
	if firstIsFile {
		typ = EntryTyp_File
	}
	newRootEnt := &Entry{
		Cid:    newRoot,
		Typ:    typ,
		Offset: 0,
		Len:    0,
	}
	f.Trees = append(f.Trees, newRootEnt)
	return InsertWithFullPath(newRootEnt, cid)
}

func InsertWithFullPath(node *Entry, cid []string) *Entry {
	if len(cid) == 0 {
		return nil
	}
	first := &Entry{
		Cid: cid[0],
		Typ: EntryTyp_Dir,
	}
	if node == nil {
		node = first
		cid = cid[1:]
	}
	if len(cid) == 0 {
		first.Typ = EntryTyp_File
	}
	return insert(node, cid)
}

func InsertIncrement(node *Entry, cid ...string) *Entry {
	if node == nil {
		return nil
	}
	if len(cid) == 0 {
		return node
	}
	ent := Entry{
		Cid: cid[0],
		Typ: EntryTyp_Dir,
	}
	node.Children = append(node.Children, &Link{Entry: &ent})
	if len(cid) == 1 {
		return &ent
	}
	return InsertIncrement(&ent, cid[1:]...)
}

// insert `CidB/CidC/CidD` into `CidA/CidB/CidC`, CidD is file and the rest are directory
// insert node with full path and return the leaf node
func insert(node *Entry, cid []string) *Entry {
	if node == nil {
		return nil
	}
	if len(cid) == 0 {
		return node
	}
	entry := &Entry{
		Cid: cid[0],
		Typ: EntryTyp_Dir,
	}
	if len(cid) == 1 {
		entry.Typ = EntryTyp_File
		node.Children = append(node.Children, &Link{Entry: entry})
		return entry
	}
	currCid := cid[0]
	for _, link := range node.Children {
		if link.Entry.Cid == currCid {
			return insert(link.Entry, cid[1:])
		}
	}
	node.Children = append(node.Children, &Link{Entry: entry})
	return insert(entry, cid[1:])
}

func Find(node *Entry, cid string) *Entry {
	if node == nil {
		return nil
	}
	if node.Cid == cid {
		return node
	}
	var children []*Entry
	for _, link := range node.Children {
		if link.Entry.Cid == cid {
			return link.Entry
		}
		children = append(children, link.Entry)
	}
	for _, child := range children {
		ans := Find(child, cid)
		if ans != nil {
			return ans
		}
	}
	return nil
}
