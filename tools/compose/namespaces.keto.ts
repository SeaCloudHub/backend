import { Namespace, SubjectSet, Context } from "@ory/keto-namespace-types"

class User implements Namespace {
    related: {
        manager: User[]
    }
}