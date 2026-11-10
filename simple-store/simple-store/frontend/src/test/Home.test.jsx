import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../api'
import Home from '../pages/Home'
import { renderApp } from './utils'

const mocks = vi.hoisted(() => ({ products: vi.fn(), categories: vi.fn() }))
vi.mock('../api', async (importOriginal) => {
  const actual = await importOriginal()
  return { ...actual, api: { ...actual.api, products: mocks.products, categories: mocks.categories } }
})

const PRODUCTS = [
  { id: 1, title: 'Backpack', price: 109.95, category: 'bags', image: '' },
  { id: 2, title: 'Gold Ring', price: 695, category: 'jewelery', image: '' },
]

describe('Home (Guest storefront)', () => {
  beforeEach(() => {
    mocks.products.mockReset()
    mocks.categories.mockReset()
    mocks.categories.mockResolvedValue(['bags', 'jewelery'])
  })

  it('shows a loading state, then the product list with prices', async () => {
    mocks.products.mockResolvedValue(PRODUCTS)
    renderApp(<Home />)

    expect(screen.getByRole('status')).toHaveTextContent('กำลังโหลดสินค้า')
    expect(await screen.findByText('Backpack')).toBeInTheDocument()
    expect(screen.getByText('Gold Ring')).toBeInTheDocument()
    expect(screen.getByText('$109.95')).toBeInTheDocument()
    expect(screen.getByText('$695.00')).toBeInTheDocument()
  })

  it('shows an empty state when there are no products', async () => {
    mocks.products.mockResolvedValue([])
    renderApp(<Home />)

    expect(await screen.findByText(/ยังไม่มีสินค้าในร้าน/)).toBeInTheDocument()
  })

  it('shows the error message and recovers when "ลองอีกครั้ง" is clicked', async () => {
    mocks.products
      .mockRejectedValueOnce(new ApiError(500, 'internal server error'))
      .mockResolvedValueOnce(PRODUCTS)
    renderApp(<Home />)

    expect(await screen.findByRole('alert')).toHaveTextContent('internal server error')
    await userEvent.click(screen.getByRole('button', { name: 'ลองอีกครั้ง' }))

    expect(await screen.findByText('Backpack')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('refetches with the category when a category chip is clicked', async () => {
    mocks.products.mockResolvedValue(PRODUCTS)
    renderApp(<Home />)

    await userEvent.click(await screen.findByRole('button', { name: 'jewelery' }))

    expect(mocks.products).toHaveBeenLastCalledWith({ category: 'jewelery', q: '' })
  })

  it('searches products by text', async () => {
    mocks.products.mockResolvedValue(PRODUCTS)
    renderApp(<Home />)
    await screen.findByText('Backpack')

    await userEvent.type(screen.getByLabelText('ค้นหาสินค้า'), 'ring')
    await userEvent.click(screen.getByRole('button', { name: 'ค้นหา' }))

    expect(mocks.products).toHaveBeenLastCalledWith({ category: '', q: 'ring' })
  })
})
