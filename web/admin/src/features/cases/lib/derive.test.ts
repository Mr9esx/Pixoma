import { describe, expect, it } from 'vitest'
import {
  deriveBindings,
  deriveInputSchema,
  normalizeOutputTypes,
  validateEditor,
  type InputFieldDraft,
  type OutputFieldDraft,
} from './derive'

const inputs: InputFieldDraft[] = [
  {
    key: 'reference',
    type: 'image',
    required: true,
    node_id: '1',
    field_path: 'image',
  },
  {
    key: 'prompt',
    type: 'string',
    required: true,
    node_id: '2',
    field_path: 'text',
  },
  {
    key: 'style',
    type: 'enum',
    required: false,
    node_id: '2',
    field_path: 'style',
    enum_values: ['anime', 'photo'],
  },
  {
    key: 'seed',
    type: 'number',
    required: false,
    node_id: '3',
    field_path: 'seed',
  },
]

const outputs: OutputFieldDraft[] = [
  { key: 'image', type: 'image', node_id: '9', index: 0 },
]

describe('derive', () => {
  it('derives bindings from field drafts', () => {
    expect(deriveBindings(inputs, outputs)).toEqual({
      inputs: [
        { key: 'reference', node_id: '1', field_path: 'image' },
        { key: 'prompt', node_id: '2', field_path: 'text' },
        { key: 'style', node_id: '2', field_path: 'style' },
        { key: 'seed', node_id: '3', field_path: 'seed' },
      ],
      outputs: [{ key: 'image', node_id: '9', index: 0 }],
    })
  })

  it('normalizes output types from bound nodes', () => {
    const outputs: OutputFieldDraft[] = [
      { key: 'image', type: 'file', node_id: 'node-a', index: 0 },
      { key: 'unbound', type: 'image', node_id: '', index: 0 },
    ]
    expect(
      normalizeOutputTypes(outputs, [
        { id: 'node-a', class_type: 'VHS_VideoCombine' },
      ])
    ).toEqual([
      { key: 'image', type: 'video', node_id: 'node-a', index: 0 },
      { key: 'unbound', type: 'image', node_id: '', index: 0 },
    ])
  })

  it('derives input_schema with types, enums and required', () => {
    expect(deriveInputSchema(inputs)).toEqual({
      type: 'object',
      additionalProperties: false,
      required: ['reference', 'prompt'],
      properties: {
        reference: { type: 'string' },
        prompt: { type: 'string' },
        style: { type: 'string', enum: ['anime', 'photo'] },
        seed: { type: 'number' },
      },
    })
  })

  it('drops empty keys from schema and bindings', () => {
    const withEmpty = [{ ...inputs[0], key: '' }]
    expect(deriveInputSchema(withEmpty)).toEqual({
      type: 'object',
      additionalProperties: false,
      required: [],
      properties: {},
    })
    expect(deriveBindings(withEmpty, []).inputs).toEqual([])
  })

  it('validates duplicates, unbound required inputs and missing outputs', () => {
    const bad: InputFieldDraft[] = [
      {
        key: 'x',
        type: 'string',
        required: true,
        node_id: '',
        field_path: '',
      },
      {
        key: 'x',
        type: 'string',
        required: false,
        node_id: '1',
        field_path: 'a',
      },
    ]
    expect(validateEditor(bad, [])).toEqual({
      duplicateKey: 'x',
      unboundInput: 'x',
      noOutput: true,
    })
    expect(validateEditor([inputs[0]], outputs)).toEqual({})
  })

  it('rejects optional inputs without a binding', () => {
    const unbound = [{ ...inputs[3], node_id: '', field_path: '' }]
    expect(validateEditor(unbound, outputs)).toEqual({
      unboundInput: 'seed',
    })
  })
})
