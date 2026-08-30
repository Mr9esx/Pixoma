import type { KnipConfig } from 'knip'

const config: KnipConfig = {
  ignore: [
    'src/components/ui/**',
    'src/components/Dither.d.ts',
    'src/components/FaultyTerminal.d.ts',
    'src/components/filters/filter-segment.tsx',
    'src/tanstack-table.d.ts',
  ],
  ignoreDependencies: ['@radix-ui/react-collapsible', 'input-otp'],
}

export default config
