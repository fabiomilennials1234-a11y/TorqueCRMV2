import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Card, CardHeader, CardTitle, CardBody, CardFooter } from '../card'

describe('Card', () => {
  it('renders full composition', () => {
    const { container } = render(
      <Card>
        <CardHeader>
          <CardTitle>Funil</CardTitle>
        </CardHeader>
        <CardBody>Corpo</CardBody>
        <CardFooter>rodapé</CardFooter>
      </Card>
    )
    expect(container).toMatchSnapshot()
  })

  it('renders bare card', () => {
    const { container } = render(<Card>content</Card>)
    expect(container).toMatchSnapshot()
  })
})
