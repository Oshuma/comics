import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Button, Dropdown, Form, Modal } from 'react-bootstrap'
import { Link } from 'react-router'
import { api, errorMessage, type Group } from '../api'
import { useFlash } from '../context'
import { Icon } from './Common'

export function GroupNameModal({
  show,
  title,
  initialName = '',
  onHide,
  onSave,
}: {
  show: boolean
  title: string
  initialName?: string
  onHide: () => void
  onSave: (name: string) => Promise<void>
}) {
  const [name, setName] = useState(initialName)
  const [saving, setSaving] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (show) setName(initialName)
  }, [show, initialName])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      await onSave(name)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal show={show} onHide={onHide} size="sm" onEntered={() => inputRef.current?.select()}>
      <Form onSubmit={submit}>
        <Modal.Header closeButton>
          <Modal.Title as="h5">{title}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form.Control
            ref={inputRef}
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Group name..."
            required
          />
        </Modal.Body>
        <Modal.Footer>
          <Button type="submit" disabled={saving || !name.trim()}>
            Save
          </Button>
          <Button variant="secondary" onClick={onHide}>
            Cancel
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  )
}

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
    <GroupNameModal
      show={editing}
      title={`Edit: ${group.name}`}
      initialName={group.name}
      onHide={() => setEditing(false)}
      onSave={rename}
    />
  )

  return { items, modal }
}
