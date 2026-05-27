/**
 * @file selectableImage.ts
 * @description Image extension with click-to-select support.
 * Clicking an image creates a NodeSelection so the image gets the
 * `.ProseMirror-selectednode` class and can be aligned/deleted via toolbar.
 */

import Image from '@tiptap/extension-image'
import { Plugin, NodeSelection } from '@tiptap/pm/state'

export const SelectableImage = Image.extend({
  addProseMirrorPlugins() {
    const parentPlugins = this.parent?.() ?? []
    return [
      ...parentPlugins,
      new Plugin({
        props: {
          handleClickOn(view, _pos, node, nodePos, _event, direct) {
            if (!direct || node.type.name !== 'image') return false
            const sel = NodeSelection.create(view.state.doc, nodePos)
            view.dispatch(view.state.tr.setSelection(sel))
            return true
          },
        },
      }),
    ]
  },
})
