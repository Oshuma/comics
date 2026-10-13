import { useState } from 'react'
import { Dropdown } from 'react-bootstrap'
import { Link } from 'react-router'
import { api, errorMessage, type Group } from '../api'
import { useFlash } from '../context'
import { Icon, NameModal } from './Common'

interface GroupActionsProps {
  group: Group
  onChanged: () => void
  onDeleted: () => void
}

// Dropdown items for managing a group, and its edit modal (render outside the menu).
export function useGroupMenu({ group, onChanged, onDeleted }: GroupActionsProps) {
  const flash = useFlash()
  const [editing, setEditing] = useState(false)

  const run = async (confirmText: string, action: () => Promise<unknown>, done: string, after: () => void) => {
    if (!window.confirm(confirmText)) return
    try {
      await action()
      flash('success', done)
      after()
    } catch (err) {
      flash('danger', errorMessage(err))
    }
  }

  const rename = async (name: string) => {
    try {
      await api.renameGroup(group.id, name)
      flash('success', 'Group updated.')
      setEditing(false)
      onChanged()
    } catch (err) {
      flash('danger', `There was a problem updating that group: ${errorMessage(err)}`)
    }
  }

  const items = (
    <>
      <Dropdown.Item as={Link} to={`/upload?group_id=${group.id}`}>
        <Icon name="cloud-upload" className="me-2" />
        Upload Comics
      </Dropdown.Item>
      <Dropdown.Item onClick={() => setEditing(true)}>
        <Icon name="pencil" className="me-2" />
        Edit Group
      </Dropdown.Item>

      {group.comicCount > 0 && (
        <>
          <Dropdown.Divider />
          <Dropdown.Item
            onClick={() =>
              run('Remove all read comics?', () => api.deleteGroupReadComics(group.id), 'Read comics removed.', onChanged)
            }
          >
            <Icon name="eye-slash" className="me-2" />
            Remove Read Comics
          </Dropdown.Item>
          <Dropdown.Item
            onClick={() =>
              run('Remove all comics in this group?', () => api.deleteGroupComics(group.id), 'Comics removed.', onChanged)
            }
          >
            <Icon name="slash-circle" className="me-2" />
            Remove All Comics
          </Dropdown.Item>
        </>
      )}

      <Dropdown.Divider />
      <Dropdown.Item
        className="text-danger"
        onClick={() => run('Delete this group?', () => api.deleteGroup(group.id), 'Group removed.', onDeleted)}
      >
        <Icon name="trash" className="me-2" />
        Delete Group
      </Dropdown.Item>
    </>
  )

  const modal = (
    <NameModal
      show={editing}
      title={`Edit: ${group.name}`}
      initialName={group.name}
      placeholder="Group name..."
      onHide={() => setEditing(false)}
      onSave={rename}
    />
  )

  return { items, modal }
}
