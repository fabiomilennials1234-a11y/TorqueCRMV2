import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { StepProgress } from '../step-progress'

const steps = ['Identidade', 'Tom', 'Skills', 'Gatilho', 'Revisão']

describe('StepProgress', () => {
  it('renders at step 1/5 (nothing completed)', () => {
    const { container } = render(<StepProgress steps={steps} currentStep={0} />)
    expect(container).toMatchSnapshot()
  })

  it('renders at step 3/5 with first two completed', () => {
    const { container } = render(
      <StepProgress steps={steps} currentStep={2} completedSteps={[0, 1]} />
    )
    expect(container).toMatchSnapshot()
  })

  it('renders at step 5/5 all completed', () => {
    const { container } = render(
      <StepProgress steps={steps} currentStep={4} completedSteps={[0, 1, 2, 3]} />
    )
    expect(container).toMatchSnapshot()
  })

  it('renders minimal variant', () => {
    const { container } = render(
      <StepProgress steps={steps} currentStep={2} completedSteps={[0, 1]} variant="minimal" />
    )
    expect(container).toMatchSnapshot()
  })
})
