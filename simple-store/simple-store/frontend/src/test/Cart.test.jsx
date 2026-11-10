import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { useCart } from '../cart'
import { renderApp } from './utils'

const A = { id: 1, title: 'Backpack', price: 10, image: '' }
const B = { id: 2, title: 'Ring', price: 2.5, image: '' }

function Probe() {
  const { items, add, setQty, remove, count, total } = useCart()
  return (
    <div>
      <button onClick={() => add(A)}>add A</button>
      <button onClick={() => add(B, 4)}>add 4 B</button>
      <button onClick={() => setQty(1, 3)}>A to 3</button>
      <button onClick={() => setQty(1, 0)}>A to 0</button>
      <button onClick={() => remove(2)}>remove B</button>
      <output data-testid="summary">{`${items.length}|${count}|${total}`}</output>
    </div>
  )
}

const summary = () => screen.getByTestId('summary').textContent

describe('cart', () => {
  it('merges the same product, and computes count and total', async () => {
    renderApp(<Probe />)

    await userEvent.click(screen.getByText('add A'))
    await userEvent.click(screen.getByText('add A'))
    await userEvent.click(screen.getByText('add 4 B'))

    expect(summary()).toBe('2|6|30') // A×2 (20) + B×4 (10)
  })

  it('updates quantity, removes lines, and drops a line when qty < 1', async () => {
    renderApp(<Probe />)
    await userEvent.click(screen.getByText('add A'))
    await userEvent.click(screen.getByText('add 4 B'))

    await userEvent.click(screen.getByText('A to 3'))
    expect(summary()).toBe('2|7|40')

    await userEvent.click(screen.getByText('remove B'))
    expect(summary()).toBe('1|3|30')

    await userEvent.click(screen.getByText('A to 0'))
    expect(summary()).toBe('0|0|0')
  })

  it('keeps the cart in localStorage so a guest keeps it after login', async () => {
    renderApp(<Probe />)
    await userEvent.click(screen.getByText('add A'))

    expect(JSON.parse(localStorage.getItem('store.cart'))).toEqual([{ id: 1, title: 'Backpack', price: 10, image: '', qty: 1 }])
  })
})
