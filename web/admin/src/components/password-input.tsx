import { SecretInput } from './secret-input'

type PasswordInputProps = Omit<
  React.InputHTMLAttributes<HTMLInputElement>,
  'type'
> & {
  ref?: React.Ref<HTMLInputElement>
}

export function PasswordInput(props: PasswordInputProps) {
  return <SecretInput {...props} />
}
