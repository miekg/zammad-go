# ticketfs

Ticketfs is a (toy) FUSE filesystem that maps the (open OR new) tickets from Zammad into a file system.
It serves as a (maybe useful?) example on how to use the zammad-go API.

Each ticket is a directory, the owner of the ticket directory is the user handling the ticket or
'nobody' when there isn't anyway. And group is the group the tickets is assigned to. The timestamps
are:

- `atime`: Ticket.LastContactAt
- `mtime`: Ticket.UpdatedAt
- `ctime`: Ticket.CreatedAt

And `ls -l` shows thus:

    drwxrwxr-x. 1 miek    sysadmin 0 Jul  3 07:00 346/

* Bits: always 775 for dirs (664 for files)
* Nr of links: 1
* User: owner of ticket (nobody if not assigned)
* Group: group of this ticket (nobody if not assigned)
* Size:size of file, or 4096 for directories
* Mtime: see above
* Name: `<ticket number>` (here 346)

Chowning the directory to a different user assigns the ticket, chgrp sets the group. The
users/groups are synced from zammad and then matched (as strings) against system users and group. To
if a user has the 'name' and a system user of 'name' exist, the user ID of the system user is used
in the filesystem. If no match is found the ID from zammad is used as-is. The same holds true for
groups. Note that this data is retrieved on start up only.

In a ticket directory you see:

- `ticket`: the original contents of the ticket (read only)
- `state`: current ticket state: `echo 'closed' > state` will close the ticket, note that if you
  close the ticket, the ticket will disappear from the filesystem
- `ID`: file with the ID and link to zammad (read only)
- `number`: file with the ticket number (readonly)
- `articles`: all "articles belonging to this ticket. This is a fifo file, see below.
- `tags`: all tags this ticket has, editing this file and removing or adding tags will
    update the tickets tag, one tag per line

Writing to:

- `articles` adds a new article using the current user (that is mapped to the Zammad-id)
- `state`: writing a valid state to it and the ticket's state will the change. Valid states
    are 'new', 'open', 'closed'

Based upon https://github.com/anacrolix/fuse/blob/master/examples/clockfs/clockfs.go
