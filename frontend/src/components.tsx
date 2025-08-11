import React from 'react'

export const Card: React.FC<{title: string, extra?: React.ReactNode, children?: React.ReactNode}> = ({title, extra, children}) => (
  <div className="card">
    <div className="row" style={{justifyContent: 'space-between'}}>
      <h3 className="title">{title}</h3>
      <div>{extra}</div>
    </div>
    {children}
  </div>
)
