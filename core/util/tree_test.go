package util

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInsertIntoEmpty(t *testing.T) {
	var cid = []string{"A"}
	entry := InsertWithFullPath(nil, cid)
	require.NotNil(t, entry)
	require.True(t, entry.Cid == "A")
	require.True(t, entry.Typ == EntryTyp_File)
}

func TestInsert(t *testing.T) {
	root := Entry{
		Cid: "A",
		Typ: EntryTyp_Dir,
		Children: []*Link{{
			Entry: &Entry{
				Cid: "B",
				Typ: EntryTyp_Dir,
				Children: []*Link{{
					Entry: &Entry{
						Cid: "C",
						Typ: EntryTyp_Dir,
					},
				}},
			},
		}},
	}
	entry := InsertWithFullPath(&root, []string{"B", "C", "D"})
	require.NotNil(t, entry)
	require.True(t, entry.Cid == "D")
	require.True(t, entry.Typ == EntryTyp_File)
	fmt.Printf("%+v", root)
}

func TestFind(t *testing.T) {
	root := Entry{
		Cid: "A",
		Typ: EntryTyp_Dir,
		Children: []*Link{{
			Entry: &Entry{
				Cid: "B",
				Typ: EntryTyp_Dir,
				Children: []*Link{{
					Entry: &Entry{
						Cid: "C",
						Typ: EntryTyp_File,
					},
				}},
			},
		}},
	}
	find := Find(&root, "C")
	require.NotNil(t, find)
	require.True(t, find.Cid == "C")
	require.True(t, find.Typ == EntryTyp_File)
	require.True(t, len(find.Children) == 0)
}
