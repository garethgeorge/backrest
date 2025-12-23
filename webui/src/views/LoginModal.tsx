import React, { useState } from "react";
import { authenticationService, setAuthToken } from "../api";
import { LoginRequestSchema } from "../../gen/ts/v1/authentication_pb";
import { useAlertApi } from "../components/Alerts";
import { create } from "@bufbuild/protobuf";
import * as m from "../paraglide/messages";
import { FormModal } from "../components/FormModal";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { InputGroup } from "@/components/ui/input-group";
import { Input, Flex, Stack } from "@chakra-ui/react";
import { LuUser, LuLock } from "react-icons/lu";

export const LoginModal = () => {
    let defaultCreds = create(LoginRequestSchema, {});
    const [username, setUsername] = useState(defaultCreds.username);
    const [password, setPassword] = useState(defaultCreds.password);
    const [loading, setLoading] = useState(false);

    const alertApi = useAlertApi()!;

    const handleSubmit = async (e?: React.FormEvent) => {
        if (e) e.preventDefault();
        
        if (!username || !password) {
             // basic validation visual feedback handled by 'required' on fields usually, 
             // but we can rely on disable logic or just let the user try.
             return;
        }

        setLoading(true);

        const loginReq = create(LoginRequestSchema, {
            username: username,
            password: password,
        });

        try {
            const loginResponse = await authenticationService.login(loginReq);
            setAuthToken(loginResponse.token);
            alertApi.success(m.login_success(), 5);
            setTimeout(() => {
                window.location.reload();
            }, 500);
        } catch (e: any) {
            alertApi.error(m.login_error() + (e.message ? e.message : "" + e), 10);
            setLoading(false);
        }
    };

    return (
        <FormModal
            isOpen={true}
            onClose={() => {}} // Non-closable
            title={m.login_title()}
            footer={
                  <Button type="submit" loading={loading} onClick={() => handleSubmit()} width="full">
                      {m.login_button()}
                  </Button>
            }
        >
            <form onSubmit={handleSubmit}>
                <Stack gap={4}>
                    <Field label="Username" required errorText={!username ? m.login_username_required() : undefined}>
                         <InputGroup startElement={<LuUser />}>
                            <Input 
                                placeholder={m.login_username_placeholder()}
                                value={username}
                                onChange={(e) => setUsername(e.target.value)}
                            />
                        </InputGroup>
                    </Field>

                    <Field label="Password" required errorText={!password ? m.login_password_required() : undefined}>
                        <InputGroup startElement={<LuLock />}>
                            <Input 
                                type="password"
                                placeholder={m.login_password_placeholder()}
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                            />
                        </InputGroup>
                    </Field>
                </Stack>
            </form>
        </FormModal>
    );
};

