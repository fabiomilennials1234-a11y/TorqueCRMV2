import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../tabs'

describe('Tabs', () => {
  it('renders three tabs with first active', () => {
    const { container } = render(
      <Tabs defaultValue="a">
        <TabsList>
          <TabsTrigger value="a">A</TabsTrigger>
          <TabsTrigger value="b">B</TabsTrigger>
          <TabsTrigger value="c">C</TabsTrigger>
        </TabsList>
        <TabsContent value="a">alpha</TabsContent>
        <TabsContent value="b">beta</TabsContent>
        <TabsContent value="c">gamma</TabsContent>
      </Tabs>
    )
    expect(container).toMatchSnapshot()
  })
})
