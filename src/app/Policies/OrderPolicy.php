<?php

namespace App\Policies;

use App\Models\Order;
use App\Models\User;

class OrderPolicy
{
    /** Admins may act on any order. */
    public function before(User $user): ?bool
    {
        return $user->isAdmin() ? true : null;
    }

    /** The buyer, or a B2B colleague from the same company, may view an order. */
    public function view(User $user, Order $order): bool
    {
        return $this->owns($user, $order);
    }

    /** Same ownership rule governs who may pay for an order. */
    public function pay(User $user, Order $order): bool
    {
        return $this->owns($user, $order);
    }

    private function owns(User $user, Order $order): bool
    {
        if ($order->user_id === $user->id) {
            return true;
        }

        return $user->isB2b()
            && $order->company_id !== null
            && $order->company_id === $user->company_id;
    }
}
