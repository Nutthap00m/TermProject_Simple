export function Loading({ label = 'กำลังโหลด...' }) {
  return (
    <div className="status" role="status">
      {label}
    </div>
  )
}

export function Empty({ children }) {
  return <div className="status empty">{children}</div>
}

export function ErrorBox({ error, onRetry }) {
  return (
    <div className="status error" role="alert">
      <p>{error?.message || 'เกิดข้อผิดพลาด'}</p>
      {onRetry && (
        <button type="button" onClick={onRetry}>
          ลองอีกครั้ง
        </button>
      )}
    </div>
  )
}
