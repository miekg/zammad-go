package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/AlessandroSechi/zammad-go"
	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	_ "github.com/anacrolix/fuse/fs/fstestutil"
	"go.science.ru.nl/log"
)

// This defines the file names we have in each ticket directory.
const (
	FileTitle    = "title"    // Title of the ticket.
	FileState    = "state"    // State of the ticket as string.
	FileID       = "ID"       // ID of the ticket (same as directory you are in) as link to Zammad.
	FileNumber   = "number"   // Number of the ticket (as link to Zammad(?)).
	FileArticles = "articles" // Number of the ticket (as link to Zammad(?)).
	FileTags     = "tags"     // Tags of the ticket.

	URL = "https://helpdesk.science.ru.nl"
)

var (
	flagToken = flag.String("t", "", "token to use for zammad authentication")
	flagURL   = flag.String("u", "", "URL for zammad")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	log.D.Set()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("zammad"),
		fuse.Subtype("ticketfs"),
		fuse.LocalVolume(),
		fuse.VolumeName("Zammad Ticket Fs"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	state(*flagToken, *flagURL)

	err = fs.Serve(c, FS{Token: *flagToken, URL: *flagURL})
	if err != nil {
		log.Fatal(err)
	}

	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

type FS struct {
	Token string
	URL   string
}

func (fs FS) Root() (fs.Node, error) {
	return &Dir{Path: "/", z: NewZammad(fs.Token, fs.URL), Tickets: &([]zammad.Ticket{})}, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	Path string // Full path leading to this directory.
	Name string // Name of this directory, empty for root

	z          *zammad.Client
	Ticket     zammad.Ticket
	Tickets    *[]zammad.Ticket // Each "ticket" dir has one ticket, the root holds multiple. This a pointer otherwise "hash of unhashable type main.Dir"
	LastUpdate time.Time        // if older than 5s we refresh from Zammad.
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0o775
	a.Gid = uint32(d.Ticket.GroupID)
	a.Uid = uint32(d.Ticket.OwnerID)
	a.Size = 12
	a.Atime = d.Ticket.LastContactAt
	a.Mtime = d.Ticket.UpdatedAt
	a.Ctime = d.Ticket.CreatedAt
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	log.Debugf("d.Lookup %s %s %q %d", d.Path, d.Name, name, d.Ticket.ID)
	// Handle the names we allow in the fs to mimic tickets, should match the names we return ReadDirAll.

	// d.Name should contain the ticket directory name
	switch name {
	case FileTitle:
		return &File{Path: path.Join(d.Path, d.Name), Name: name, ReadOnly: true, Ticket: d.Ticket, LastUpdate: d.LastUpdate, z: d.z}, nil
	case FileState:
		return &File{Path: path.Join(d.Path, d.Name), Name: name, Ticket: d.Ticket, LastUpdate: d.LastUpdate, z: d.z}, nil
	case FileID:
		return &File{Path: path.Join(d.Path, d.Name), Name: name, ReadOnly: true, Ticket: d.Ticket, LastUpdate: d.LastUpdate, z: d.z}, nil
	case FileNumber:
		return &File{Path: path.Join(d.Path, d.Name), Name: name, ReadOnly: true, Ticket: d.Ticket, LastUpdate: d.LastUpdate, z: d.z}, nil
	case FileArticles:
		return &File{Path: path.Join(d.Path, d.Name), Name: name, Ticket: d.Ticket, LastUpdate: d.LastUpdate, z: d.z}, nil
	case FileTags:
		return &File{Path: path.Join(d.Path, d.Name), Name: name, Ticket: d.Ticket, LastUpdate: d.LastUpdate, z: d.z}, nil
	}

	// this should be /, check d.LastUpdate, and return the ticket from the ticket slice
	if x := time.Now().Sub(d.LastUpdate).Seconds(); x > 5 {
		ticket, err := d.z.TicketShow(int(ParseUint(name)))
		if err != nil {
			return nil, err
		}
		if ticket.ID == 0 {
			return nil, syscall.ENOENT
		}
		return &Dir{Path: path.Join(d.Path, d.Name), Name: name, Ticket: ticket, z: d.z, LastUpdate: time.Now()}, nil
	}
	// use name (ticket ID) to return the correct ticket
	nameID := ParseUint(name)
	if nameID == 0 || d.Tickets == nil {
		return nil, syscall.ENOENT
	}
	for _, ticket := range *d.Tickets {
		if ticket.ID == int(nameID) {
			return &Dir{Path: path.Join(d.Path, d.Name), Name: name, Ticket: ticket, z: d.z, LastUpdate: time.Now()}, nil
		}
	}
	return nil, syscall.ENOENT
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	log.Debugf("d.ReadDirAll %s %s", d.Path, d.Name)
	fullpath := path.Join(d.Path, d.Name)
	switch {
	case fullpath == "/":
		if x := time.Now().Sub(d.LastUpdate).Seconds(); x > 5 {
			log.Debugf("Older then 5s (%2.2f), requerying", x)
			tickets, err := d.z.TicketSearch(url.QueryEscape("state.name:(new OR open OR pending)"), 10000)
			if err != nil {
				return nil, err
			}
			*d.Tickets = tickets
			d.LastUpdate = time.Now()
		}
		entries := make([]fuse.Dirent, len(*d.Tickets))
		for i := range *d.Tickets {
			entries[i].Name = fmt.Sprintf("%d", (*d.Tickets)[i].ID)
			entries[i].Type = fuse.DT_Dir
		}
		return entries, nil

	case d.Path == "/" && ParseUint(d.Name) != 0: // Specific ticket
		if x := time.Now().Sub(d.LastUpdate).Seconds(); x > 5 {
			log.Debugf("Older then 5s (%2.2f), requerying", x)
			ticket, err := d.z.TicketShow(int(ParseUint(d.Name)))
			if err != nil {
				return nil, err
			}
			if ticket.ID == 0 {
				return nil, syscall.ENOENT
			}
			d.Ticket = ticket
			d.LastUpdate = time.Now()
		}
		// Files that exist in every ticket directory
		entries := []fuse.Dirent{
			{Name: FileTitle, Type: fuse.DT_File},
			{Name: FileState, Type: fuse.DT_File},
			{Name: FileID, Type: fuse.DT_File},
			{Name: FileNumber, Type: fuse.DT_File},
			{Name: FileArticles, Type: fuse.DT_File},
			{Name: FileTags, Type: fuse.DT_File},
		}
		return entries, nil
	}
	return nil, nil
}

type File struct {
	Path     string // see Dir
	Name     string // see Dir
	ReadOnly bool
	zammad.Ticket
	tags       *[]zammad.Tag
	articles   *[]zammad.TicketArticle
	z          *zammad.Client
	LastUpdate time.Time
}

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Debugf("f.Attr %s %s %d", f.Path, f.Name, f.Ticket.ID)
	if x := time.Now().Sub(f.LastUpdate).Seconds(); x > 5 {
		log.Debugf("f.Attr: older then 5s (%2.2f), requerying", x)
		ticket, err := f.z.TicketShow(f.Ticket.ID)
		if err != nil {
			return err
		}
		if ticket.ID == 0 {
			return syscall.ENOENT
		}
		f.Ticket = ticket
		f.LastUpdate = time.Now()
	}

	a.Mode = 0o664
	if f.ReadOnly {
		a.Mode = 0o444
	}
	a.Gid = uint32(f.Ticket.GroupID)
	a.Uid = uint32(f.Ticket.OwnerID)
	a.Atime = f.Ticket.LastContactAt
	a.Mtime = f.Ticket.UpdatedAt
	a.Ctime = f.Ticket.CreatedAt

	s := 0
	switch f.Name {
	case FileTitle:
		s = len(f.Ticket.Title) + 1
	case FileState:
		s = len(TicketState[f.Ticket.StateID])
	case FileID:
		s = len(fmt.Sprintf("%s/#ticket/zoom/%d\n", URL, f.Ticket.ID))
	case FileNumber:
		s = len([]byte(f.Ticket.Number + "\n"))
	case FileArticles:
		a, _ := f.ArticleContent()
		s = len(a)
	case FileTags:
		a, _ := f.TagContent()
		s = len(a)
	}
	a.Size = uint64(s)
	return nil
}

func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	log.Debugf("f.ReadAll %s %s %d", f.Path, f.Name, f.Ticket.ID)
	switch f.Name {
	case FileTitle:
		return []byte(f.Ticket.Title + "\n"), nil
	case FileState:
		return []byte(TicketState[f.Ticket.StateID]), nil
	case FileID:
		return []byte(fmt.Sprintf("%s/#ticket/zoom/%d\n", URL, f.Ticket.ID)), nil
	case FileNumber:
		return []byte(f.Ticket.Number + "\n"), nil
	case FileArticles:
		return f.ArticleContent()
	case FileTags:
		return f.TagContent()
	}
	return nil, nil
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	if f.Name != FileArticles {
		return syscall.ENOSYS
	}

	ta := ArticleWrite(f.Ticket, req)
	_, err := f.z.TicketArticleCreate(ta)
	if err != nil {
		log.Errorf("Failed to write article: %s", err)
		return fmt.Errorf("Failed to write article: %s", err)
	}
	resp.Size = len(req.Data)
	return nil
}
