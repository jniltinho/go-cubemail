/**
 * @file fontFamily.ts
 * @description TipTap extension that adds a fontFamily attribute to the TextStyle mark.
 * Follows the same pattern as @tiptap/extension-color: registers a global attribute
 * on the 'textStyle' mark so that font-family can be applied as inline style.
 * All identifiers and comments are in English per project policy.
 */

import { Extension } from '@tiptap/core'

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    fontFamily: {
      setFontFamily: (fontFamily: string) => ReturnType
      unsetFontFamily: () => ReturnType
    }
  }
}

/** Registers fontFamily as a textStyle attribute and exposes set/unset commands. */
export const FontFamily = Extension.create({
  name: 'fontFamily',

  addOptions() {
    return {
      types: ['textStyle'],
    }
  },

  addGlobalAttributes() {
    return [
      {
        types: this.options.types,
        attributes: {
          fontFamily: {
            default: null,
            parseHTML: el => el.style.fontFamily?.replace(/['"]/g, '') || null,
            renderHTML: attrs => {
              if (!attrs.fontFamily) return {}
              return { style: `font-family: ${attrs.fontFamily}` }
            },
          },
        },
      },
    ]
  },

  addCommands() {
    return {
      setFontFamily:
        (fontFamily: string) =>
        ({ chain }) =>
          chain().setMark('textStyle', { fontFamily }).run(),

      unsetFontFamily:
        () =>
        ({ chain }) =>
          chain()
            .setMark('textStyle', { fontFamily: null })
            .removeEmptyTextStyle()
            .run(),
    }
  },
})
