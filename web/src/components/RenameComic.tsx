import { useState } from 'react'
import { api, errorMessage, type Comic } from '../api'
import { useFlash } from '../context'
import { NameModal } from './Common'

// Rename action for a comic, and its modal (render outside any menu).
export function useRenameComic(comic: Pick<Comic, 'id' | 'name'>, onRenamed: () => void) {
  const flash = useFlash()
  const [show, setShow] = useState(false)

  const save = async (name: string) => {
    try {
      await api.renameComic(comic.id, name)
      flash('success', 'Comic renamed.')
      setShow(false)
      onRenamed()
    } catch (err) {
      flash('danger', `There was a problem renaming that comic: ${errorMessage(err)}`)
    }
  }

  const modal = (
    <NameModal
      show={show}
      title={`Rename: ${comic.name}`}
      initialName={comic.name}
      placeholder="Comic name..."
      onHide={() => setShow(false)}
      onSave={save}
    />
  )

  return { open: () => setShow(true), modal }
}
