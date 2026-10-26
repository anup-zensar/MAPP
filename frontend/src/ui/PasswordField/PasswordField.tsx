import { TextField } from '../TextField/TextField';
import type { TextFieldProps } from '../TextField/TextField';

export type PasswordFieldProps = Omit<TextFieldProps, 'type'>;

/** A TextField for a password: what is typed is hidden until the eye button shows it. */
export function PasswordField(props: PasswordFieldProps) {
  return <TextField {...props} type="password" />;
}
