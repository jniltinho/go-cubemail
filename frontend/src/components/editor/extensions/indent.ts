/**
 * @file indent.ts
 * @description Indent/outdent extension for paragraphs and headings.
 * Applies margin-left in 40px steps (up to 8 levels).
 * For list items use sinkListItem/liftListItem instead.
 */

import { Extension } from '@tiptap/core'
import type { Node } from '@tiptap/pm/model'

const STEP = 40
const MAX_LEVEL = 8

const BLOCK_TYPES = ['paragraph', 'heading', 'blockquote']

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    indent: {
      indent: () => ReturnType
      outdent: () => ReturnType
    }
  }
}

export const Indent = Extension.create({
  name: 'indent',

  addGlobalAttributes() {
    return [
      {
        types: BLOCK_TYPES,
        attributes: {
          indent: {
            default: 0,
            parseHTML: (el: HTMLElement) => {
              const ml = parseInt(el.style.marginLeft || '0', 10)
              return isNaN(ml) ? 0 : Math.round(ml / STEP)
            },
            renderHTML: (attrs: Record<string, unknown>) => {
              const level = (attrs.indent as number) || 0
              if (!level) return {}
              return { style: `margin-left: ${level * STEP}px` }
            },
          },
        },
      },
    ]
  },

  addCommands() {
    return {
      indent: () => ({ tr, state, dispatch }) => {
        const { from, to } = state.selection
        state.doc.nodesBetween(from, to, (node: Node, pos: number) => {
          if (!BLOCK_TYPES.includes(node.type.name)) return
          const cur = (node.attrs.indent as number) || 0
          const next = Math.min(cur + 1, MAX_LEVEL)
          if (next !== cur) tr.setNodeMarkup(pos, undefined, { ...node.attrs, indent: next })
        })
        if (dispatch) dispatch(tr)
        return true
      },

      outdent: () => ({ tr, state, dispatch }) => {
        const { from, to } = state.selection
        state.doc.nodesBetween(from, to, (node: Node, pos: number) => {
          if (!BLOCK_TYPES.includes(node.type.name)) return
          const cur = (node.attrs.indent as number) || 0
          const next = Math.max(cur - 1, 0)
          if (next !== cur) tr.setNodeMarkup(pos, undefined, { ...node.attrs, indent: next })
        })
        if (dispatch) dispatch(tr)
        return true
      },
    }
  },
})
